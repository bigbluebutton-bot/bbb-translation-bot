import socket


# Connect to localhost:5000 and send a message
def connect():
    try:
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        s.connect(('localhost', 5000))
        s.sendall(b'Hello, world')
        s.close()
    except:
        print("Connection failed")

connect()
