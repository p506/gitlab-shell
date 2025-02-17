package main_test

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

	"github.com/mikesmitty/edkey"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

var (
	sshdPath = ""
)

func init() {
	rootDir := rootDir()
	sshdPath = filepath.Join(rootDir, "bin", "gitlab-sshd")

	if _, err := os.Stat(sshdPath); os.IsNotExist(err) {
		panic(fmt.Errorf("cannot find executable %s. Please run 'make compile'", sshdPath))
	}
}

func rootDir() string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		panic(fmt.Errorf("rootDir: calling runtime.Caller failed"))
	}

	return filepath.Join(filepath.Dir(currentFile), "..", "..")
}

func successAPI(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("gitlab-api-mock: received request: %s %s", r.Method, r.RequestURI)
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.EscapedPath() {
		case "/api/v4/internal/authorized_keys":
			fmt.Fprintf(w, `{"id":1, "key":"%s"}`, r.FormValue("key"))
		case "/api/v4/internal/discover":
			fmt.Fprint(w, `{"id": 1000, "name": "Test User", "username": "test-user"}`)
		default:
			t.Log("Unexpected request to successAPI!")
			t.FailNow()
		}
	})
}

func genServerConfig(gitlabUrl, hostKeyPath string) []byte {
	return []byte(`---
user: "git"
log_file: ""
log_format: json
secret: "0123456789abcdef"
gitlab_url: "` + gitlabUrl + `"
sshd:
  listen: "127.0.0.1:0"
  web_listen: ""
  host_key_files:
    - "` + hostKeyPath + `"`)
}

func buildClient(t *testing.T, addr string, hostKey ed25519.PublicKey) *ssh.Client {
	t.Helper()

	pubKey, err := ssh.NewPublicKey(hostKey)
	require.NoError(t, err)

	_, clientPrivKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	clientSigner, err := ssh.NewSignerFromKey(clientPrivKey)
	require.NoError(t, err)

	client, err := ssh.Dial("tcp", addr, &ssh.ClientConfig{
		User:            "git",
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(clientSigner)},
		HostKeyCallback: ssh.FixedHostKey(pubKey),
	})
	require.NoError(t, err)

	t.Cleanup(func() { client.Close() })

	return client
}

func configureSSHD(t *testing.T, apiServer string) (string, ed25519.PublicKey) {
	t.Helper()

	dir, err := ioutil.TempDir("", "gitlab-sshd-acceptance-test-")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })

	configFile := filepath.Join(dir, "config.yml")
	hostKeyFile := filepath.Join(dir, "hostkey")

	pub, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	configFileData := genServerConfig(apiServer, hostKeyFile)
	require.NoError(t, ioutil.WriteFile(configFile, configFileData, 0644))

	block := &pem.Block{Type: "OPENSSH PRIVATE KEY", Bytes: edkey.MarshalED25519PrivateKey(priv)}
	hostKeyData := pem.EncodeToMemory(block)
	require.NoError(t, ioutil.WriteFile(hostKeyFile, hostKeyData, 0400))

	return dir, pub
}

func startSSHD(t *testing.T, dir string) string {
	t.Helper()

	// We need to scan the first few lines of stderr to get the listen address.
	// Once we've learned it, we'll start a goroutine to copy everything to
	// the real stderr
	pr, pw := io.Pipe()
	t.Cleanup(func() { pr.Close() })
	t.Cleanup(func() { pw.Close() })

	scanner := bufio.NewScanner(pr)
	extractor := regexp.MustCompile(`msg="Listening on ([0-9a-f\[\]\.:]+)"`)

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, sshdPath, "-config-dir", dir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = pw
	require.NoError(t, cmd.Start())
	t.Logf("gitlab-sshd: Start(): success")
	t.Cleanup(func() { t.Logf("gitlab-sshd: Wait(): %v", cmd.Wait()) })
	t.Cleanup(cancel)

	var listenAddr string
	for scanner.Scan() {
		if matches := extractor.FindSubmatch(scanner.Bytes()); len(matches) == 2 {
			listenAddr = string(matches[1])
			break
		}
	}
	require.NotEmpty(t, listenAddr, "Couldn't extract listen address from gitlab-sshd")

	go io.Copy(os.Stderr, pr)

	return listenAddr
}

// Starts an instance of gitlab-sshd with the given arguments, returning an SSH
// client already connected to it
func runSSHD(t *testing.T, apiHandler http.Handler) *ssh.Client {
	t.Helper()

	// Set up a stub gitlab server
	apiServer := httptest.NewServer(apiHandler)
	t.Logf("gitlab-api-mock: started: url=%q", apiServer.URL)
	t.Cleanup(func() {
		apiServer.Close()
		t.Logf("gitlab-api-mock: closed")
	})

	dir, hostKey := configureSSHD(t, apiServer.URL)
	listenAddr := startSSHD(t, dir)

	return buildClient(t, listenAddr, hostKey)
}

func TestDiscoverSuccess(t *testing.T) {
	client := runSSHD(t, successAPI(t))

	session, err := client.NewSession()
	require.NoError(t, err)
	defer session.Close()

	output, err := session.Output("discover")
	require.NoError(t, err)
	require.Equal(t, "Welcome to GitLab, @test-user!\n", string(output))
}
