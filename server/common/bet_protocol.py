from common.socket_utils import read_n_bytes, send_all
from common.utils import Bet

def read_bet_from_socket(client_sock):
        try:
            fields_length = read_n_bytes(client_sock, 6)
        except OSError:
            raise OSError("Socket closed while reading field")
        
        if len(fields_length) < 6:
            raise ProtocolError("Received incomplete header: lengths missing")
        
        bet_fields = []
        for i in range(6):
            field_len = int.from_bytes(fields_length[i], byteorder='big')
            try:
                field = read_n_bytes(client_sock, field_len)
            except OSError:
                raise OSError("Socket closed while reading field")
            bet_fields.append(field.decode('utf-8'))

        if len(bet_fields) < 6:
            raise ProtocolError("Received incomplete bet: fields missing")

        return Bet(bet_fields[0], bet_fields[1], bet_fields[2], bet_fields[3], bet_fields[4], bet_fields[5])

def send_ack(socket):
    send_all(socket, b'\x00')

def send_nack(socket):
    send_all(socket, b'\x01')

class ProtocolError(Exception):
    """ Custom exception for protocol errors """
