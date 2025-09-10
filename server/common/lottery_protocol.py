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
        offset = 0
        bets = []
        status = "success"

        try:
            # Read the length of the incoming batch (2 bytes) -> to know how many bytes to read
            batch_len = read_n_bytes(client_sock, BATCH_HEADER_SIZE)
            length = int(batch_len[0])<<8 | int(batch_len[1])
        except ProtocolError:
            raise ProtocolError("Fail to read from socket")
        
        # Read the whole batch -> read all in one read call
        batch = read_n_bytes(client_sock, length)

        # Read bets
        while offset < length:     
            # Not enough bytes for a full bet header       
            if offset + TOTAL_FIELDS_BET > length:
                status = "fail"
                break

            # Read the length of each field (6 bytes) and update offset
            fields_length = []
            for i in range(TOTAL_FIELDS_BET):
                fields_length.append(batch[offset + i])
            offset += TOTAL_FIELDS_BET

            # Not enough bytes for the full bet according to the lengths read
            if offset + sum(fields_length) > len(batch):
                status = "fail"
                break

            # Read each field according to its length and update offset
            bet_fields, offset = read_single_bet(fields_length, batch, offset)

            bets.append(Bet(bet_fields[0], bet_fields[1], bet_fields[2], bet_fields[3], bet_fields[4], bet_fields[5]))

        if len(bets) > 0:
            logging.info(f'action: apuesta_recibida | result: {status} | cantidad: {len(bets)}')
        return bets

def read_single_bet(fields_length, batch, offset):
    """
    Reads all necessary fields to complete a bet
    """
    bet_fields = []
    for field_len in fields_length:
        field = batch[offset: offset+field_len]
        bet_fields.append(field.decode('utf-8'))
        offset += field_len

    return bet_fields, offset

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

