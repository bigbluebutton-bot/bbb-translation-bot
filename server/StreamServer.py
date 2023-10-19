import inspect
import logging
import threading
import time
import json
import TCPserver as TCPserver
import UDPserver as UDPserver

# EXAMPLE USAGE
SECRET_TOKEN = "your_secret_token"




class EventHandler:
    """Class responsible for managing callbacks."""

    def __init__(self):
        logging.debug("Initializing EventHandler.")
        self._callbacks = {}
        self._event_lock = threading.Lock()

    def add_event(self, callback):
        """Add a new event callback and return its unique ID."""
        with self._event_lock:
            event_id = callback
            logging.debug(f"Adding event with ID: {event_id}")
            self._callbacks[event_id] = callback
            return event_id

    def remove_event(self, event_id):
        """Remove an event callback using its ID."""
        logging.debug(f"Removing event with ID: {event_id}")
        with self._event_lock:
            self._callbacks.pop(event_id, None)

    def emit(self, *args):
        """Trigger all the registered callbacks with the provided arguments."""
        threads = []  # To keep track of the threads

        with self._event_lock:
            for event_id, callback in self._callbacks.items():
                logging.debug(f"Emitting event with ID: {event_id}")
                # Wrap the callback execution in a thread
                t = threading.Thread(target=callback, args=args)
                threads.append(t)
                t.start()

        # Wait for all threads to finish
        for t in threads:
            t.join()


class StreamServer:

    def __init__(self, host, tcpport, udpportmin, udpportmax, secrettoken="", encryption=False, timeout=5):
        self.host = host
        self.tcpport = tcpport
        self.udpportmin = udpportmin
        self.udpportmax = udpportmax
        self.udpports = set()
        self.secrettoken = secrettoken
        self.encryption = encryption
        self.timeout = timeout

        self.udpports_lock = threading.Lock()
        self.udpservers_lock = threading.Lock()

        self._connected_callbacks = EventHandler()
        self.tcpserver = TCPserver.Server(host, tcpport, timeout, encryption, 5, 1, secrettoken)
        self.udpserver_tcpclient_list = []


    def start(self):
        self.tcpserver.on_connected(lambda c: self._on_tcp_connected(c))
        self.tcpserver.start()

    def _on_tcp_connected(self, client):
        # select a udp port between udpportmin and udpportmax which isnt in use (in udpports)
        selectedtport = 0
        with self.udpports_lock:
            for port in range(self.udpportmin, self.udpportmax):
                if not port in self.udpports:
                    selectedtport = port
                    break

        if selectedtport == 0:
            logging.error("No free UDP port available!")
            client.stop()
            return
        
        # add selected port to udpports
        with self.udpports_lock:
            self.udpports.add(selectedtport)

        # get aes_key and aes_initkey
        aes_key = client.client_key
        aes_initkey = client.client_initkey

        udpencryption = False
        if self.encryption:
            udpencryption = True

        # create new udp server and add it to the map
        udpserver = UDPserver.Server(self.host, selectedtport, udpencryption, aes_key, aes_initkey)

        # add udpserver to map
        with self.udpservers_lock:
            streamclient = self.StreamClient(self._remove_client, udpserver, client)
            self.udpserver_tcpclient_list.append(streamclient)

        udpserver.start()

        # send udp port to client in json format
        jsondata = json.dumps({"type": "init_udpaddr", "msg": {"udp": {"host": self.host, "port": selectedtport}}}).encode()
        client.send(jsondata)

        # emit on_connected event
        self._connected_callbacks.emit(streamclient)

    def stop(self):
        self.tcpserver.stop()

        for udpserver_tcpclient in self.udpserver_tcpclient_list:
            udpserver_tcpclient.stop()

        self.udpserver_tcpclient_list = []

        while self.tcpserver.active_clients_count > 0:  # Wait for all clients to disconnect
            logging.info(f"Waiting for {self.tcpserver.active_clients_count} clients to disconnect...")
            time.sleep(1)
        logging.info("Server stopped.")


    def _remove_client(self, streamclient):
        """Private method to remove a client from the server's client list."""
        logging.debug(f"Removing client: {streamclient.tcpclient.addr}")
        with self.udpservers_lock:
            self.udpserver_tcpclient_list.remove(streamclient)
        with self.udpports_lock:
            self.udpports.discard(streamclient.udpserver.port)


    def on_connected(self, callback):
        """Register a callback for when a client connects."""
        logging.debug("Registering 'connected' event.")
        # Get the number of parameters the callback has
        num_params = len(inspect.signature(callback).parameters)

        if num_params != 1:
            logging.error(f"Invalid number of parameters for 'connected' event. Expected 1, got {num_params}.")
            return
        
        return self._connected_callbacks.add_event(callback)

    def remove_connected_event(self, event_id):
        """Remove the connected event callback using its ID."""
        logging.debug("Removing 'connected' event.")
        self._connected_callbacks.remove_event(event_id)




    class StreamClient():
        def __init__(self, on_remove, udpserver, tcpclient):
            self._on_remove = on_remove
            self.udpserver = udpserver
            self.tcpclient = tcpclient

            # stop udp server if disconnected or timeout from tcp server
            self.tcpclient.on_event("disconnected", lambda c: self.stop())
            self.tcpclient.on_event("timeout", lambda c: self.stop())

        def send(self, data):
            jsondata = json.dumps({"type": "msg", "msg": data}).encode()
            self.udpserver.send(data)

        def on_message(self, callback):
            self.tcpclient.on_event("message", callback)

        def stop(self):
            self.tcpclient.stop()
            self.udpserver.stop()
            self._on_remove(self)







def on_connected(streamclient):
    print(f"Connected by {streamclient.tcpclient.addr}")
    streamclient.on_message(lambda c, d: print(f"Received message from {c.addr}: {d}"))


def main():
    srv = StreamServer("localhost", 5000, 5001, 5099, SECRET_TOKEN, 4096, 5)

    srv.on_connected(on_connected)

    srv.start()

    try:
        while True:  # Keep the server running until a keyboard interrupt
            time.sleep(1)
    except KeyboardInterrupt:
        logging.info("Stopping server...")
        srv.stop()

        logging.info("THE END")

if __name__ == '__main__':
    main()