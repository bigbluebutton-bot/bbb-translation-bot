import inspect
import logging
import threading
import time
import json
import TCPserver as TCPserver
import UDPserver as UDPserver
import Event as event
EventHandler = event.EventHandler

logging.basicConfig(level=logging.DEBUG)


class Client:
    def __init__(self, on_remove, tcpclient, udpclient):
        self._on_remove = on_remove
        self._tcpclient = tcpclient
        self._udpclient = udpclient

        # if the tcp client disconnects or timeouts, remove the client
        self._tcpclient.on_event("disconnected", lambda c: self.stop())
        self._tcpclient.on_event("timeout", lambda c: self.stop())

    def udp_address(self):
        """Return the clients UDP address."""
        return self._udpclient.address()
    
    def tcp_address(self):
        """Return the clients TCP address."""
        return self._tcpclient.address()
    
    def stop(self):
        logging.debug(f"Stopping client {self._tcpclient.address()}, {self._udpclient.address()}.")
        self._tcpclient.stop()
        self._udpclient.stop()
        self._on_remove(self)



    def send_message(self, message):
        """Send a TCP message to the client."""
        self._tcpclient.send(message)



    def on_tcp_message(self, callback):
        """Register a new TCP message callback."""
        return self._tcpclient.on_event("message", lambda c, d: callback(self, d))
    
    def remove_on_tcp_message(self, callback):
        """Remove a TCP message callback using its ID."""
        return self._tcpclient.remove_event("message", callback)
    
    
    def on_udp_message(self, callback):
        """Register a new UDP message callback."""
        return self._udpclient.on_event("message", lambda c, d: callback(self, d))
    
    def remove_on_udp_message(self, callback):
        """Remove a UDP message callback using its ID."""
        return self._udpclient.remove_event("message", callback)
    
    
    def on_disconnected(self, callback):
        """Register a new disconnected callback."""
        return self._tcpclient.on_event("disconnected", lambda c: callback(self))
    
    def remove_on_disconnected(self, callback):
        """Remove a disconnected callback using its ID."""
        return self._tcpclient.remove_event("disconnected", callback)
    
    
    def on_timeout(self, callback):
        """Register a new timeout callback."""
        return self._tcpclient.on_event("timeout", lambda c: callback(self))
    
    def remove_on_timeout(self, callback):
        """Remove a timeout callback using its ID."""
        return self._tcpclient.remove_event("timeout", callback)
    




class Server:
    def __init__(self, host, tcpport, udpport, secrettoken="", encryption=False, timeout=5, maxclients=10, buffer_size=1024):
        self._host = host
        self._tcpport = tcpport
        self._udpport = udpport
        self._secrettoken = secrettoken
        self._encryption = encryption
        self._timeout = timeout
        self._maxclients = maxclients
        self._buffer_size = buffer_size

        self._clients = {} # {tcpaddr: client}

        self._tcpserver = TCPserver.Server(self._host, self._tcpport, self._timeout, self._encryption, 5, self._maxclients, self._secrettoken, self._buffer_size)
        self._udpserver = UDPserver.Server(self._host, self._udpport, self._encryption, self._buffer_size)

        self._connected_callbacks = EventHandler()

        # event. tcp client on connect:
        # 1. add client to udpserver whitelist
        # 2. add tcp and udp client to self._clients
        # 3. send udp server address to client
        # 4. emit event connected
        def _on_tcp_connected(tcpclient):
            # 1. add client to udpserver whitelist
            clienthost = tcpclient.address()[0]
            aes_key = tcpclient.client_key
            aes_initkey = tcpclient.client_initkey
            udpclient = self._udpserver.add_client(clienthost, aes_key, aes_initkey)

            # 2. add tcp and udp client to self._clients
            client = Client(self._remove_client, tcpclient, udpclient)
            self._clients[tcpclient.address()] = client

            # 3. send udp server address to client
            udpencryption = udpclient._encryption
            jsondata = json.dumps({"type": "init_udpaddr", "msg": {"udp": {"host": self._host, "port": self._udpport, "encryption": udpencryption}}}).encode()
            tcpclient.send(jsondata)

            # 4. emit event connected
            self._connected_callbacks.emit(client)

        self._tcpserver.on_connected(_on_tcp_connected)


    def start(self):
        """Start the server."""
        if self._tcpserver._running:
            logging.warning("TCP server is already running.")
            return
        if self._udpserver._running:
            logging.warning("UDP server is already running.")
            return
        self._tcpserver.start()
        self._udpserver.start()

    def stop(self):
        """Stop the server."""
        self._tcpserver.stop()
        self._udpserver.stop()


    def on_connected(self, callback):
        """Register a new connected callback."""
        # Get the number of parameters the callback has
        num_params = len(inspect.signature(callback).parameters)

        if num_params != 1:
            logging.error(f"Invalid number of parameters for 'connected' event. Expected 1, got {num_params}.")
            return
        
        return self._connected_callbacks.add_event(callback)
    
    def remove_on_connected(self, callback):
        """Remove a connected callback using its ID."""
        return self._connected_callbacks.remove_event(callback)
    

    def _remove_client(self, client):
        """Remove a client from the server's client list."""
        logging.debug(f"Removing client: {client.tcp_address()}, {client.udp_address()}")
        self._clients.pop(client.tcp_address(), None)


# EXAMPLE USAGE
SECRET_TOKEN = "your_secret_token"

def main():
    srv = Server("127.0.0.1", 5000, 5001, SECRET_TOKEN, 4096, 5, 10, 1024)

    def _on_connected(client):
        print(f"Client connected: {client.tcp_address()}, {client.udp_address()}")
        client.send_message(b"Hello from server! Connected")

        def _on_tcp_message(c, message):
            print(f"Received TCP message from client: {c.tcp_address()}: {message}")
            client.send_message(b"Hello from server! TCP")

        def _on_udp_message(c, message):
            print(f"Received UDP message from client: {c.udp_address()}: {message}")
            client.send_message(b"Hello from server! UDP")

        def _on_disconnected(c):
            print(f"Client disconnected: {c.tcp_address()}, {c.udp_address()}")

        client.on_tcp_message(_on_tcp_message)
        client.on_udp_message(_on_udp_message)
        client.on_disconnected(_on_disconnected)

    srv.on_connected(_on_connected)

    srv.start()

    try:
        while True:  # Keep the server running until a keyboard interrupt
            time.sleep(1)
    except KeyboardInterrupt:
        logging.info("Stopping server...")
        srv.stop()

        logging.info("Server stopped.")
        logging.info("THE END")

if __name__ == "__main__":
    main()