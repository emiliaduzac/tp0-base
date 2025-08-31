def read_n_bytes(client_sock, n):
    data = bytearray()
    while len(data) < n:
        packet = client_sock.recv(n - len(data))
        if not packet:
            raise OSError("Socket closed while reading data")
        data.extend(packet)
    return data

def send_all(client_sock, msg):
    """
    Send data to a socket
    """
    total_sent = 0
    while total_sent < len(msg):
        sent = client_sock.send(msg[total_sent:])
        total_sent += sent