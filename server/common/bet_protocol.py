from common.socket_utils import read_n_bytes, send_all
from common.utils import Bet
import logging

IS_LAST_SIZE = 1
BATCH_HEADER_SIZE = 2
HEADER_SIZE = 6
TOTAL_FIELDS_BET = 6

def read_bets_from_socket(client_sock):
        """
        Read a whole batch of bets from a socket according to the protocol.
        First byte indicates if its the last batch (1) or not (0).
        Next two bytes indicate the length of the batch.
        Then, each bet is read according to the protocol until the read limit is reached (according to 
        the total length of the batch indicated before).
        """
        total_read = 0
        bets = []
        ok_bets = 0
        status = "success"

        # Read if its the last batch (1 byte)
        is_last = read_n_bytes(client_sock, IS_LAST_SIZE)[0] == 1
        # Read the length of the incoming batch (2 bytes)
        batch_len = read_n_bytes(client_sock, BATCH_HEADER_SIZE)
        # If it could not read these bytes, return empty bets and True to close the connection
        if is_last is None or batch_len is None:
            return bets, True
        length = int(batch_len[0])<<8 | int(batch_len[1])
        
        # Read the whole batch
        while total_read < length:
            # Read next bet
            fields_length = read_n_bytes(client_sock, HEADER_SIZE)
            if len(fields_length) < TOTAL_FIELDS_BET:
                status = "fail"
                continue
            
            bet_fields = read_single_bet(fields_length, client_sock)
            if bet_fields is None:
                status = "fail"
                break

            if len(bet_fields) < TOTAL_FIELDS_BET:
                status = "fail"
                continue

            bets.append(Bet(bet_fields[0], bet_fields[1], bet_fields[2], bet_fields[3], bet_fields[4], bet_fields[5]))
            total_read += HEADER_SIZE + sum(fields_length)
            ok_bets += 1

        logging.info(f'action: apuesta_recibida | result: {status} | cantidad: {ok_bets}')
        return bets, is_last

def read_single_bet(fields_length, client_sock):
    bet_fields = []
    for i in range(TOTAL_FIELDS_BET):
        field_len = fields_length[i]
        field = read_n_bytes(client_sock, field_len)
        if field is None:
            return None
        bet_fields.append(field.decode('utf-8'))

    return bet_fields

def send_ack(socket):
    send_all(socket, b'\x00')

def send_nack(socket):
    send_all(socket, b'\x01')

class ProtocolError(Exception):
    """ Custom exception for protocol errors """
