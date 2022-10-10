# MP2

<h3>  The System Diagram </h3>

<h3> The Code Flow: </h3>

Server - Main Function -> reads argument from command line (os.Args) -> listen on tcp channel on port -> establish TCP link -> create channel "newConn" to handle incoming client connection -> run a thread to run handleConnection() with dictionary and current connection -> 
<br/>
infinite for loop -> accept connection from client -> append connection to channel -> handleConnection()
<br/>
<br/>

d -> dictionary [ key: username of client / value : net.Conn ]
<br/>
<br/>
handleConnection() -> add sender username and connection to dictionary -> infinite loop -> look up receiver -> if receiver found -> encode content -> write to reciever connection
<br/>
<br/>

Lookup() -> takes username and look up username in the dictionary "d" -> return connection if found and nill otherwise 
<br/>
<br/>
Client - Main function -> reads host, port, and username from user -> Dial to the port -> 
write to TCP connection establish intial message to server to establish itself as sender -> create thread to recieve message rec()
<br/>
infinite loop  -> decode message from sender ->  create thread for sending message using send() -> read from freshInput until program exits
<br/>
<br/>
rec() -> print sender and content -> if content from server is exit then end all threads and pass ending message to server -> else print reciever username and content
<br/>
<br/>
send() -> read user message in the format "send dest cotent" -> package into message object -> encode message -> write into server connection -> ends process if user input exit

<h3> The Code Flow: </h3>
Server creation:
./go run server/server.go [port]
<br/>
<br/>
Client creation:
./go run client/client.go [host number] [port] [username]
<br/>
<br/>
To send a message from client1 to client2:
send client2 [message]
<br/>
<br/>
To end a client / server:
exit
<br/>
<br/>
Any commands typed into the standard input without either of those two commands will be ignored.
<br/>
<br/>
Sending a message to an invalid client will result in reciever not found error