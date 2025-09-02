import signal
import socket
import logging

from common.utils import store_bets, load_bets, has_won
from common.bet_protocol import read_from_socket, ProtocolError, send_ack, send_nack, handle_batch, send_winners
from common.protocol_utils import getAgency

class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self.clients_sending = listen_backlog # no se si es esto pero mientras, TODO
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
        print("Client connected")
        while True:
            try:
                # queda bloqueado escuchando del cliente. Nunca va a poder manejar varios a la vez
                is_batch, op_code = read_from_socket(client_sock)
                if is_batch:
                    print("Recibi batch")
                    self.receive_batches(op_code, client_sock)
                else:
                    print("Recibi request winners")
                    self.receive_request_winners(client_sock)
            except Exception:
                client_sock.close()
                break

        # close connection if it wasn't closed yet. If it was closed, nothing happens
        client_sock.close()
            
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

    def receive_batches(self, op_code, addr, client_sock):
        while True:
            try:
                bets, more_batchs_coming = handle_batch(client_sock, op_code)
                store_bets(bets)
                print("Stored bets & sent ack")
                send_ack(client_sock)

                addr = client_sock.getpeername()
                logging.info(f'action: send_message | result: success | ip: {addr[0]}')
                
                if not more_batchs_coming:
                    self.clients_sending -= 1
                    if self.clients_sending == 0:
                        logging.info("action: sorteo | result: success")
                        self.send_winners_to_waiting_clients()
                    return
                
            except OSError as e:

                logging.error(f"action: receive_message | result: fail | error: {e}")
                return
            
            except ProtocolError as e:
                send_nack(client_sock)
                return
            
    def receive_request_winners(self, client_sock):
        try:
            agency = getAgency(client_sock)
        except ProtocolError as e:
            send_nack(client_sock)
            return

        if self.clients_sending == 0:
            winners = self.get_winners(agency)
            send_winners(client_sock, winners)
        else:
            addr = client_sock.getpeername()
            self.clients_waiting_winners[agency] = addr[0]
            client_sock.close() 

    def send_winners_to_waiting_clients(self):
        for agency, ip in self.clients_waiting_winners.items:
            winners = self.get_winners(agency)
            client_sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            try:
                send_winners(client_sock, winners)
            except OSError as e:
                logging.error(f"action: send_winners | result: fail | error: {e}")
            finally:
                client_sock.close()

    def get_winners(agency):
        bets = load_bets()
        winners = []
        for bet in bets:
            if bet.agency == agency and has_won(bet):
                winners.append(bet.document)
        return winners
        
    