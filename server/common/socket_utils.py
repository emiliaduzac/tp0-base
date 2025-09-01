def read_n_bytes(client_sock, n):
    """
    Reads exactly n bytes from a socket avoiding short reads
    """
    data = bytearray()
    while len(data) < n:
        packet = client_sock.recv(n - len(data))
        if not packet:
            return None
        data.extend(packet)
    return data

def send_all(client_sock, msg):
    """
    Send data to a socket avoiding short writes
    """
    total_sent = 0
    while total_sent < len(msg):
        sent = client_sock.send(msg[total_sent:])
        total_sent += sent
