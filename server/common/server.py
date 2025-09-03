import signal
import socket
import logging
import threading

from common.utils import store_bets, load_bets, has_won
from common.lottery_protocol import ProtocolError, send_ack, send_nack, read_bets_from_socket, send_winners, read_n_bytes
from common.protocol_utils import OpCodeReq

class Server:
    def __init__(self, port, listen_backlog, total_clients):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self._clients_sending = int(total_clients)
        self._clients_waiting_winners = {}
        self._running = True
        self._bets_file_lock = threading.Lock()
        self._winners_lock = threading.Lock()

    def run(self):
        """
        Dummy Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communucation
        finishes, servers starts to accept new connections again
        """
        # set handlers for graceful shutdown
        signal.signal(signal.SIGTERM, self.__handle_shutdown) # Termination signal
        signal.signal(signal.SIGINT, self.__handle_shutdown) # Interrupt from keyboard

        while self._running:
            try:
                client_sock = self.__accept_new_connection()
                t = threading.Thread(target=self.__handle_client_connection, args=(client_sock,))
                t.start()
                # self.__handle_client_connection(client_sock)
            except OSError as e:
                logging.error(f"action: receive_message | result: fail | error: {e}")
                break
        t.join()

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
                op_code = read_n_bytes(client_sock, 1)[0]

                if op_code == OpCodeReq.OC_BATCHS.value:
                    self.__handle_batches(client_sock)

                elif op_code == OpCodeReq.OC_END.value:
                    # concu
                    self.__handle_end(client_sock)
                    break
                
                else:
                    # unkown op code
                    send_nack(client_sock)

            except OSError as e:
                logging.error(f"action: receive_message | result: fail | error: {e}")
                break
            
            except ProtocolError as e:
                send_nack(client_sock)
                logging.error(f"action: receive_message | result: fail | error: {e}")
                break
            
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
    
    def __handle_shutdown(self, signum, frame):
        """ Handle graceful shutdown of the server. """
        logging.debug(f"action: shutdown | result: in_progress | signal: {signum}")  
        self._running = False
        self.close()
        logging.debug(f"action: shutdown | result: success | signal: {signum}")

    def __handle_batches(self, client_sock):
        """ Handle the reception of a batch of bets from a client. Stores the bets in the bets file. """
        bets = read_bets_from_socket(client_sock)
        addr = client_sock.getpeername()

        if bets != None:
            with self._bets_file_lock:
                store_bets(bets)
            send_ack(client_sock)
            logging.info(f'action: send_message | result: success | ip: {addr[0]}')
            
    def __handle_end(self, client_sock):
        """ Handle the end of the communication with a client. Checks if all clients are done in order to 
        find the lottery winners and send them to each client. """
        agency = read_n_bytes(client_sock, 1)[0]

        # TO DO: manejar concurrencia en datos compartidos
        with self._winners_lock:
            self._clients_sending -= 1
            self._clients_waiting_winners[agency] = client_sock

            # Verify if all clients sent their bets to find the loterry winners
            if self._clients_sending == 0:
                logging.info(f"action: sorteo | result: success")
                all_winners = self.__get_winners()

                for act_agency, sock in self._clients_waiting_winners.items():
                    winners = all_winners.get(int(act_agency), [])
                    send_winners(sock, winners)
                    sock.close()
                    self._running = False

    def __get_winners(self):
        """ Returns a dictionary with the winners of each agency. Takes a lock of the file while reading it. """
        with self._bets_file_lock:
            bets = list(load_bets())

        winners = {}
        for bet in bets:
            if has_won(bet):
                winners[bet.agency] = winners.get(bet.agency, []) + [bet.document]
        return winners