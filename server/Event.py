import logging
import threading


class EventHandler:
    """Class responsible for managing callbacks."""

    def __init__(self):
        logging.debug("Initializing EventHandler.")
        self._callbacks = {}
        self._event_lock = threading.Lock()

    def add_event(self, callback):
        """Add a new event callback and return its unique ID."""
        with self._event_lock:
            event_id = callback
            logging.debug(f"Adding event with ID: {event_id}")
            self._callbacks[event_id] = callback
            return event_id

    def remove_event(self, event_id):
        """Remove an event callback using its ID."""
        logging.debug(f"Removing event with ID: {event_id}")
        with self._event_lock:
            self._callbacks.pop(event_id, None)

    def emit(self, *args):
        """Trigger all the registered callbacks with the provided arguments."""
        threads = []  # To keep track of the threads

        with self._event_lock:
            for event_id, callback in self._callbacks.items():
                logging.debug(f"Emitting event with ID: {event_id}")
                # Wrap the callback execution in a thread
                t = threading.Thread(target=callback, args=args)
                threads.append(t)
                t.start()

        # Wait for all threads to finish
        for t in threads:
            t.join()