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
            msg = read_socket(client_sock)
            addr = client_sock.getpeername()
            logging.info(f'action: apuesta_recibida | result: success | ip: {addr[0]} | msg: {msg}')
            
            bet_fields = msg.split(',')
            bet = Bet(bet_fields[0], bet_fields[1], bet_fields[2], bet_fields[3], bet_fields[4], bet_fields[5])
            store_bets([bet])
            logging.info(f'action: apuesta_almacenada | result: success | dni: {bet_fields[3]} | numero: {bet_fields[5]}')

            send_socket(client_sock, msg)
            logging.info(f'action: send_message | result: success | ip: {addr[0]} | msg: {msg}')

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


def read_socket(socket):
    """
    Read data from a socket until a newline character is found
    """
    b = b''
    while not b.endswith(b'\n'):
        chunk = socket.recv(1024)
        if not chunk:
            raise OSError("Client disconnected")
        b += chunk
    return b.decode('utf-8').rstrip('\n')

def send_socket(socket, msg):
    """
    Send data to a socket
    """
    total_sent = 0
    msg_as_bytes = f'{msg}\n'.encode('utf-8')
    while total_sent < len(msg_as_bytes):
        sent = socket.send(msg_as_bytes[total_sent:])
        total_sent += sent