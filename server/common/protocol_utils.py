from enum import Enum

IS_LAST_SIZE = 1
BATCH_HEADER_SIZE = 2
HEADER_SIZE = 6
TOTAL_FIELDS_BET = 6
END_OF_BATCHS = 0

class OpCodeResp(Enum):
    OC_ACK = 0x00
    OC_NACK = 0x01
    OC_WINNERS = 0x02

class OpCodeReq(Enum):
    OC_BATCHS = 0
    OC_END = 1

def read_n_bytes(client_sock, n):
    """
    Reads exactly n bytes from a socket avoiding short reads
    """
    data = bytearray()
    while len(data) < n:
        packet = client_sock.recv(n - len(data))
        if not packet:
            raise ProtocolError(f"Fail to read from socket")
        data.extend(packet)

    return data

def send_all_bytes(client_sock, msg):
    """
    Send data to a socket avoiding short writes
    """
    total_sent = 0
    while total_sent < len(msg):
        sent = client_sock.send(msg[total_sent:])
        total_sent += sent

class ProtocolError(Exception):
    """ Custom exception for protocol errors """
