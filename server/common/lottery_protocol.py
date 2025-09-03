from common.protocol_utils import read_n_bytes, send_all_bytes, ProtocolError, HEADER_SIZE, BATCH_HEADER_SIZE, TOTAL_FIELDS_BET, OpCodeResp
from common.utils import Bet
import logging

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
        status = "success"

        try:
            # Read the length of the incoming batch (2 bytes)
            batch_len = read_n_bytes(client_sock, BATCH_HEADER_SIZE)
            length = int(batch_len[0])<<8 | int(batch_len[1])
        except ProtocolError as e:
            raise ProtocolError("Fail to read from socket")
        
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

        if len(bets) > 0:
            logging.info(f'action: apuesta_recibida | result: {status} | cantidad: {len(bets)}')
        return bets

def read_single_bet(fields_length, client_sock):
    """
    Reads all necessary fields to complete a bet
    """
    bet_fields = []
    for i in range(TOTAL_FIELDS_BET):
        field_len = fields_length[i]
        field = read_n_bytes(client_sock, field_len)
        if field is None:
            return None
        bet_fields.append(field.decode('utf-8'))

    return bet_fields

def send_winners(client_sock, winners):
    """
    Notify a client with the winners of their agency
    """
    winners_msg = bytearray()
    for winner in winners:
        body = winner.encode('utf-8')
        winners_msg.extend(bytes([len(body)]))
        winners_msg.extend(body)

    msg = bytearray()
    msg.append(OpCodeResp.OC_WINNERS.value)
    msg.extend(bytes([(len(winners_msg) >> 8) & 0xFF, len(winners_msg) & 0xFF]))
    msg.extend(winners_msg)
    send_all_bytes(client_sock, msg)

def send_ack(socket):
    send_all_bytes(socket, bytes([OpCodeResp.OC_ACK.value]))

def send_nack(socket):
    send_all_bytes(socket, bytes([OpCodeResp.OC_NACK.value]))

