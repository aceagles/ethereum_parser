# Ethereum Observer
This observer collectes blocks from an ethereum endpoint and searches them for subscribed addresses.
The matching transactions are stored in the transaction store. An interface which in this example stores the transactions to memory.
The observer implements the Parser interface which can be used to connect it to a notification system in a broader system.