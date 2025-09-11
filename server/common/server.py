import signal
import socket
import logging
import threading

from common.utils import store_bets, load_bets, has_won
from common.lottery_protocol import ProtocolError, send_ack, send_nack, read_bets_from_socket, send_winners, read_n_bytes
from common.protocol_utils import OpCodeReq, AGENCY_HEADER, END_LENGTH

class Server:
    def __init__(self, port, listen_backlog, total_clients):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)

        # Winners related attributes
        self._clients_sending = int(total_clients)
        self._barrier = threading.Barrier(int(total_clients))

        # Attribute to verify if server is running
        self._running = True

        # Locks for thread safety -> bets file locks, finished clients counter lock, sockets lock
        self._bets_file_lock = threading.Lock()
        self._winners_lock = threading.Lock()
        self._socket_lock = threading.Lock()

        # threads
        self._cli_threads = []

        # client sockets to close in case of shutdown
        self._client_sockets = set() 


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

        try:
            self._server_socket.settimeout(2.0)
            while self._running:
                try:
                    client_sock = self.__accept_new_connection()
                    # verify if any thread has finished to join it and remove it from the list
                    self.__check_finished_threads()
                    with self._socket_lock:
                        self._client_sockets.add(client_sock)
                    # create a new thread to handle the client
                    t = threading.Thread(target=self.__handle_client_connection, args=(client_sock,))
                    t.start()
                    self._cli_threads.append(t)

                except socket.timeout:
                    # If timeout occurs, check again if server is still running
                    continue

                except OSError as e:
                    logging.error(f"action: receive_message | result: fail | error: {e}")
                    break

        finally:
            self.__clean_resources()

    
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

            
    def __handle_client_connection(self, client_sock):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        try:
            while self._running:
                try:
                    op_code = read_n_bytes(client_sock, 1)[0]

                    if op_code == OpCodeReq.OC_BATCHS.value:
                        self.__handle_batches(client_sock)

                    elif op_code == OpCodeReq.OC_END.value:
                        self.__handle_end(client_sock)
                        break
                    
                    else:
                        # unkown op code
                        logging.error("action: receive_message | result: fail | error: Unknown message")
                        send_nack(client_sock)

                except OSError as e:
                    logging.error(f"action: receive_message | result: fail | error: {e}")
                    break
                
                except ProtocolError as e:
                    send_nack(client_sock)
                    logging.error(f"action: receive_message | result: fail | error: {e}")
                    break
        finally:
            try:
                client_sock.close()
                with self._socket_lock:
                    self._client_sockets.remove(client_sock)
                logging.debug("action: close_connection | result: success")
            except Exception as e:
                logging.error(f"action: close_connection | result: fail | error: {e}")


    def __handle_batches(self, client_sock):
        """ Handle the reception of a batch of bets from a client. Stores the bets in the bets file. """
        bets = read_bets_from_socket(client_sock)
        addr = client_sock.getpeername()

        if len(bets) > 0:
            with self._bets_file_lock:
                store_bets(bets)
            send_ack(client_sock)
            logging.info(f'action: send_message | result: success | ip: {addr[0]}')
            

    def __handle_end(self, client_sock):
        """ Handle the end of the communication with a client. Checks if all clients are done in order to 
        find the lottery winners and send them to each client. """
        agency = read_n_bytes(client_sock, AGENCY_HEADER)[0]

        with self._winners_lock:
            self._clients_sending -= 1

        self._barrier.wait()

        agency_winners = self.__get_agency_winners(agency)
        logging.info("action: sorteo | result: success")
        send_winners(client_sock, agency_winners)


    def __get_agency_winners(self, agency):
        """ Returns a dictionary with the winners of each agency. Takes a lock of the file while reading it. """
        with self._bets_file_lock: # hace falta el lock si voy a leer solo las de mi agencia?
            winners = []
            for bet in load_bets():
                if has_won(bet) and bet.agency == agency:
                    winners.append(bet.document)
            return winners

    
    def __join_threads(self):
        for t in self._cli_threads:
            t.join()
            logging.debug("action: join_client_thread | result: success")

    
    def __check_finished_threads(self):
        threads_to_remove = []
        
        for t in self._cli_threads:
            if not t.is_alive():
                t.join()
                threads_to_remove.append(t)
                logging.debug("action: join_client_thread | result: success")

        for t in threads_to_remove:
            self._cli_threads.remove(t)


    def __close_cli_sockets(self):
        with self._socket_lock:
            sockets = self._client_sockets.copy()
            self._client_sockets.clear()

        for sock in sockets:
            try:
                sock.close()
                logging.debug("action: close_client_socket | result: success")
            except Exception as e:
                logging.error(f"action: close_client_socket | result: fail | error: {e}")
        

    def __clean_resources(self):
        """ Clean up resources used by the server. """
        try:
            self._server_socket.close()
            logging.debug("action: close_server_socket | result: success")  
            self.__join_threads()
            self.__close_cli_sockets()

        except Exception as e:
            logging.error(f"action: close_resources | result: fail | error: {e}")

    
    def __handle_shutdown(self, signum, frame):
        """ Handle graceful shutdown of the server. """
        logging.debug(f"action: shutdown | result: in_progress | signal: {signum}")
        self._running = False
        self.__clean_resources()
        logging.debug(f"action: shutdown | result: success | signal: {signum}")