import signal
import socket
import logging

from common.utils import store_bets, load_bets, has_won
from common.bet_protocol import ProtocolError, send_ack, send_nack, read_bets_from_socket, send_winners

class Server:
    def __init__(self, port, listen_backlog, total_clients):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self.clients_sending = int(total_clients)
        self.clients_waiting_winners = {}
        self._running = True

    def run(self):
        """
        Dummy Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communucation
        finishes, servers starts to accept new connections again
        """
        # set handlers for graceful shutdown
        signal.signal(signal.SIGTERM, self.handle_shutdown) # Termination signal
        signal.signal(signal.SIGINT, self.handle_shutdown) # Interrupt from keyboard

        while self._running:
            try:
                client_sock = self.__accept_new_connection()
                self.__handle_client_connection(client_sock)
            except OSError as e:
                logging.error(f"action: receive_message | result: fail | error: {e}")
                break

    def close(self):
        self._server_socket.close()
        logging.debug(f"action: close_socket | result: success")  

    def __handle_client_connection(self, client_sock):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """

        while True:
            try:
                bets, more_batchs_coming, agency = read_bets_from_socket(client_sock)
                addr = client_sock.getpeername()
                
                store_bets(bets)
                send_ack(client_sock)
                logging.info(f'action: send_message | result: success | ip: {addr[0]}')
                
                if not more_batchs_coming and agency is not None:
                    self.clients_sending -= 1
                    self.clients_waiting_winners[agency] = client_sock
                    logging.info(f"action: sorteo | result: success")

                    if self.clients_sending == 0:
                        # notify all clients waiting for winners
                        all_winners = get_winners()
                        for act_agency, sock in self.clients_waiting_winners.items():
                            winners = all_winners.get(int(act_agency), [])
                            send_winners(sock, winners)
                            sock.close()
                    break

            except OSError as e:
                logging.error(f"action: receive_message | result: fail | error: {e}")
                break
            
            except ProtocolError as e:
                send_nack(client_sock)
                break
                #logging.error(f"action: receive_bet_message | result: fail | error: {e}")
            
            
    def __accept_new_connection(self):
        """
        Accept new connections

        Function blocks until a connection to a client is made.
        Then connection created is printed and returned
        """

        # Connection arrived
        logging.info('action: accept_connections | result: in_progress')
        c, addr = self._server_socket.accept()
        logging.info(f'action: accept_connections | result: success | ip: {addr[0]}')
        return c
    
    def handle_shutdown(self, signum, frame):
        """
        Handle graceful shutdown of the server
        """
        logging.debug(f"action: shutdown | result: in_progress | signal: {signum}")  
        self._running = False
        self.close()
        logging.debug(f"action: shutdown | result: success | signal: {signum}")

def get_winners():
    bets = load_bets()
    winners = {}
    for bet in bets:
        if has_won(bet):
            winners[bet.agency] = winners.get(bet.agency, []) + [bet.document]
    return winners