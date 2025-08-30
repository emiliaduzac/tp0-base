import signal
import socket
import logging

from common.utils import store_bets, Bet

class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
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
        try:
            # TODO: Modify the receive to avoid short-reads
            msg = client_sock.recv(1024).rstrip().decode('utf-8')
            addr = client_sock.getpeername()
            logging.info(f'action: apuesta_recibida | result: success | msg: {msg}')
            
            bet_list = msg.split(',')
            bet = Bet(bet_list[0], bet_list[1], bet_list[2], bet_list[3], bet_list[4], bet_list[5])
            store_bets([bet])
            
            # TODO: Modify the send to avoid short-writes
            client_sock.send("{}\n".format(msg).encode('utf-8'))

        except OSError as e:
            logging.error(f"action: receive_message | result: fail | error: {e}")
        finally:
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
