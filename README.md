# MP2

A TCP based private chatroom

<h2> File Structure: </h2>

This program consists of two files:
- `server.go`
- `client.go`

<h2> server.go Overview </h2>

This file is responsible for running the chatroom server that each client process will connect to. The file takes a port number to listen on as a command line argument. In an endless loop, this program accepts new connections and then launches a new goroutine handleConnection() for each one. Each routine is passed the new net.Connection object itself, as well as a global dictioanry containing all active connections. 

<h3> connections object </h3>

This object contains a map that is used to track all connections, as well as a mutex to prevent race conditions.

<h3> Receiving Messages </h3>

When a thread receives a message from a client, it checks for the recipients username in the global dictionary. If the name exists, the message is forwarded by the same thread over the corresponding net.Connection object from the dictonary. If the name is not found, a "not found" message is returned to the client. Otherwise, if the content of the message indicates that client is exiting, then the connection is removed from the dictionary and the routine returns.

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




