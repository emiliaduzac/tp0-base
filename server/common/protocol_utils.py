from enum import Enum

OPCODE = 1
BATCH_HEADER_SIZE = 2
HEADER_SIZE = 6
TOTAL_FIELDS_BET = 6
MORE_BATCHS_COMING = 0

class OpCodeResp(Enum):
    OC_ACK = 0x00
    OC_NACK = 0x01
    OC_WINNERS = 0x02

class OpCodeReq(Enum):
    OC_MORE_BATCHS = 0
    OC_LAST_BATCH = 1
    OC_ASK_WINNERS = 2

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

def getAgency(client_sock):
    ag = read_n_bytes(client_sock, 1)
    if ag is None:
        raise ProtocolError("Fail to read agency from socket")
    return int(ag[0])