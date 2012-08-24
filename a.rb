require 'socket'

def report_error_to_master(local, e)
  local.write "FAIL"
end

def run_action(socket, action)
  eval action
  socket.write "OK"
rescue Exception => e
  report_error_to_master(socket, e)
end

def notify_newly_loaded_files
end

def go(identifier="")
  # okay, so I ahve this FD that I can use to send data to the master.
  fd = ENV['ZEUS_MASTER_FD'].to_i
  master = UNIXSocket.for_fd(fd)

  # I need to give the master a way to talk to me exclusively
  local, remote = UNIXSocket.pair(:DGRAM)
  master.send_io(remote)

  # Now I need to tell the master about my PID and ID
  local.write %Q|{"pid": #{Process.pid}, "identifier": #{identifier.inspect}}|

  # So now we have to wait for the master to identify us and send our action.
  # We'll run the action right away and then report status to the master.
  action = local.recv(65536) # TODO: just chunk this?
  run_action(local, action)

  # the master wants to know about the files that running the action caused us to load.
  Thread.new { notify_newly_loaded_files }

  # We are now 'connected'. From this point, we may receive requests to fork.
  loop do
    new_identifier = local.recv(1024)
    # if new_identifier =~ /^command:/
    #   fork { command(new_identifier) }
    # else
    #   fork { go(new_identifier) }
    # end
  end

end

__FILE__ == $0 and go()