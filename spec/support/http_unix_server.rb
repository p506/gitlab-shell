require 'webrick'

# like WEBrick::HTTPServer, but listens on UNIX socket
class HTTPUNIXServer < WEBrick::HTTPServer
  def initialize(config = {})
    null_log = WEBrick::Log.new(IO::NULL, 7)

    super(config.merge(Logger: null_log, AccessLog: null_log))
  end

  def listen(address, _port)
    socket = Socket.unix_server_socket(address)
    socket.autoclose = false
    server = UNIXServer.for_fd(socket.fileno)
    socket.close
    @listeners << server
  end

  def cleanup_shutdown_pipe(shutdown_pipe)
    @shutdown_pipe = nil
    return unless shutdown_pipe

    super(shutdown_pipe)
  end
end
