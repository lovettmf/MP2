# MP2

A TCP based private chatroom

<h2> File Structure: </h2>

This program consists of two files:
- `server.go`
- `client.go`

<h2> server.go Overview </h2>

This file is responsible for running the chatroom server that each client process will connect to. The file takes a port number to listen on as a command line argument. In an endless loop, this program accepts new connections and then launches a new goroutine handleConnection() for each one. Each routine is passed the new net.Connection object itself, as well as a global dictioanry containing all active connections. It is expected that each routine will receive an initial message used only to associate the client username with a net.Connection object. 

<h3> connections struct </h3>

This struct contains a map that is used to track all connections, as well as a mutex to prevent race conditions.

<h3> Message struct </h3>

This struct is what will be sent over TCP. Each one contains a sender, a recipient, and message content.

<h3> Receiving Messages </h3>

When a thread receives a message from a client, it checks for the recipients username in the global map. If the name exists, the message is forwarded by the same thread over the corresponding net.Connection object from the dictonary. If the name is not found, a "not found" message is returned to the client. Otherwise, if the content of the message indicates that client is exiting, then the connection is removed from the map and the routine returns.

<h3> Exiting the server </h3>

A routine running handleExit() is launched before the main function enters its endless loop. This endlessly accepts terminal input until "exit" is entered. The main routine is then alerted via channel, which sends an exit message to each connected client before returning and ending the program. 

<h3> Usage </h3>

To launch a server from the main project folder: 
`go run ./server/server.go [PORT]`
<br></br>
To exit the server type 
`exit`
<br></br>
Example: 
```
./server/server.go 1234
>> exit
```
<br></br>
<h2> client.go Overview </h2>

This file is responsible for running client programs that can connect to a server to send messages to other clients. The file takes the IP (loaclhost in this case) and port number of the desired server, as well as a username as command line arguments. The main function will automatically intiate the connection with the server and send an initial message so that the server can identify the username associated with this client. 

<h3> Receiving Messages </h3>
 
 Before the main functon enters an endless loop it launches rec() in a goroutine. This will endlessly wait for new messages and print them to the terminal. If this routine receives an exit message from the server it will alert the main routine via channel and exit. 
 
 <h3> Sending Messages </h3>
 
The main routine will infinitely launch the send() function in a new routine (assuming the exit channel isn't filled), thus each time a user enters something, a new routine must be launched to gather further input. This is particularly useful because if the user enters erroneous input, nothing appears to happen. When the user inputs a message and destination, this is dumped into a Message struct and sent to the server and a new send routine is launched. If the user inputs "exit" the main routine is alerted via exit channel, and it sends a message to the server alerting it before returning and ending the program. 

<h3> Usage </h3>

To launch a client process from the main project folder: 
`go run ./client/client.go localhost [PORT] [USERNAME]`
<br></br>
To send a message: 
`send [USERNAME] [MESSAGE]`
<br></br>
To exit the server type 
`exit`
<br></br>
Example of sending, sending to a client that doesn't exist, receiving, and exiting. Assume a client with username 'tommy' already exists: 
```
./client/client.go localhost 1234 lovettmf
>> send tommy hello
>> send x testing
>> From Server: x not found
>> From tommy: hi back
>> exit
connection closed
```
<br></br>
<h2> Copied/Referenced Code </h2>

- https://madflojo.medium.com/making-golang-packages-thread-safe-bb9592aeab92
This set of helper functions allow global map access by accessing the struct containing it via pointer. By using a mutex when updating/accessing the map, it prevents simultaneous read/writes from different routines which would not be safe. The user of defer when unlocking the mutex is also novel to us, as it ensures the operations are definitely completed before the lock is released. 

- https://gist.github.com/MilosSimic/ae7fe8d70866e89dbd6e84d86dc8d8d5
This method for reading/writing over TCP utilizes the encoding/gob package. Put simply, gob serialized data types, including structs. This is particulalry useful for communicating over TCP which is done with binary streams. In gob encoding, types are described by a specific number. Although the decoder knows nothing about the contents it is receiving, it can still efficiently unmarshall it into the correct struct/data type. 




