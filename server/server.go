package main

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

// Message contains sender, destination, and content information to be sent between client and server
type Message struct {
	To      string
	From    string
	Content string
}

// Structure that contains a map of active tcp connections and a mutex to prevent race conditions
type connections struct {
	mu            sync.Mutex
	connectionMap map[string]net.Conn
}

//The following map helper functions are derived from: https://madflojo.medium.com/making-golang-packages-thread-safe-bb9592aeab92
//Their purpose is to manipulate the map of connections shared by server threads without inciting race conditions

//-----Start map helper functions-----

// Creates and returns a pointer to a new connections object
func New() (*connections, error) {

	//Pointer to new connections object
	d := &connections{}
	// Create new map within object
	d.connectionMap = make(map[string]net.Conn)

	return d, nil //should never produce an error
}

// Adds an entry to the map of connections using a pointer to the connection object
func (d *connections) Add(user string, c net.Conn) error {

	//Lock mutex and defer the unlock so it is not unlocked until the other operation is completely
	d.mu.Lock()
	defer d.mu.Unlock()
	d.connectionMap[user] = c

	//Need not return anything since the map is accessed via pointer
	return nil
}

// Called when a client closes a connection, this deletes an entry in the map
func (d *connections) Delete(user string) error {

	//Like Add, defer the unlock to prevent accidental premature access
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.connectionMap, user)

	return nil
}

// This function checks to see if a username is in the map and returns the corresponding net.Conn object
// Should the username not exist, the function returns "not found"
func (d *connections) Lookup(user string) (net.Conn, string) {

	d.mu.Lock()
	defer d.mu.Unlock()
	c, ok := d.connectionMap[user] //Ok will be nil if the username does not exist in the map

	if ok {
		return c, "valid"
	}

	return nil, "not found"
}

//-----End map helper functions-----

func handleConnection(c net.Conn, d *connections) {

	//This function is run in a new Goroutine every time a new connection is accepted.
	//It sends/receives messages from its client and delivers said messages to their designated recipients.

	//Some of the following code used to encode, send, and receive over TCP is derived from:
	//https://gist.github.com/MilosSimic/ae7fe8d70866e89dbd6e84d86dc8d8d5

	//An array of bytes to be used later as a buffer to send/receive Message objects
	temp := make([]byte, 500)

	//This block reads the first message sent automatically by every new client and put it into the temp array
	//Its purpose is to acquire the client's username
	_, err := c.Read(temp)
	if err != nil {
		fmt.Println(err)
		return
	}

	//The array of bytes received becomes a buffer that can be encoded
	tempBuff := bytes.NewBuffer(temp)
	//An empty Message struct for the incoming message to be dumped into
	tempMsg := new(Message)

	//Creates a new gob Decoder object
	gobObj := gob.NewDecoder(tempBuff)
	//Decodes the gob stream that was read into the buffer and offloads it into the empty Message struct
	gobObj.Decode(tempMsg)

	//Access the received message and add the username/net.Conn object to the map
	d.Add(tempMsg.From, c)

	for {

		//Repeat the above receiving process in an endless loop and handle the message contents

		_, err = c.Read(temp)
		if err != nil {
			fmt.Println(err)
			return
		}

		tempBuff = bytes.NewBuffer(temp)
		tempMsg = new(Message)

		gobObj = gob.NewDecoder(tempBuff)
		gobObj.Decode(tempMsg)

		//Lookup the connection object of the Message's destination client
		dest, status := d.Lookup(tempMsg.To)

		//If the recipient is not in the map send the original client a "not found" message
		if status == "not found" {

			//Create message for client
			notFound := Message{To: tempMsg.From, From: "Server", Content: tempMsg.To + " not found\n"}

			//Create a new byte buffer
			//Note that when sending, a byte buffer need not be converted for this purpose
			binBuff := new(bytes.Buffer)

			//Creates a new gob Encoder object using the byte buffer
			gobObject := gob.NewEncoder(binBuff)
			//Encodes the message into a gob stream then leads it into the byte buffer
			gobObject.Encode(notFound)

			//The encoded byte buffer is written over the TCP connection to the client
			c.Write(binBuff.Bytes())
			continue
		}

		//If the client sends an exit message
		if tempMsg.Content == "exiting" {

			//Remove the connection from the map
			d.Delete(tempMsg.From)
			//Close the connection
			c.Close()
			//This routine returns and ends
			return
		} else { //Any other message is forwarded accordingly

			binBuff := new(bytes.Buffer)

			gobObject := gob.NewEncoder(binBuff)
			gobObject.Encode(tempMsg)

			//Write to the destination using the connection that was looked up earlier
			dest.Write(binBuff.Bytes())
		}
	}

	c.Close()
}

// This function is run endlessly in a Goroutine to listen for a server-side exit command
func handleExit(exit chan int) {

	for {

		//Endlessly read from the standard input
		reader := bufio.NewReader(os.Stdin)
		fmt.Print(">> ")
		text, _ := reader.ReadString('\n')
		input := strings.Split(text, "\n")

		//If the input is any form of "exit" the channel is used to inform the main thread and this one returns
		//Note that any erroneous input will have no effect and the user will simply be prompted again
		if strings.ToLower(input[0]) == "exit" {
			exit <- 1
			return
		}
	}
}

func main() {

	//Some of the following code used to establish TCP connections is derived from:
	//https://www.linode.com/docs/guides/developing-udp-and-tcp-clients-and-servers-in-go/

	//Reads the command line argument for a port number to listen on
	arguments := os.Args
	if len(arguments) == 1 {
		fmt.Println("Please provide a port number!")
		return
	}

	//Start listening on the given port
	PORT := ":" + arguments[1]
	l, err := net.Listen("tcp", PORT)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer l.Close()

	//Channels that indicate an exit input or new client connection, respectively
	exit := make(chan int, 1)
	newConn := make(chan net.Conn, 1)

	//Launch a Goroutine to await a user exit command
	go handleExit(exit)

	//Create and get a pointer to a connections struct
	//This will hold the usernames/net.Conns of all current TCP connections
	d, err := New()

	for {

		//Pass the listener to a Goroutine that will accept new connections
		go func(l net.Listener) {
			for {
				//Endless attempt to accept connection
				c, err := l.Accept()
				if err != nil {
					newConn <- nil //nothing is put in the channel upon an error
					return
				}

				//Fill the channel with the accepted net.Conn object
				newConn <- c
			}
		}(l)

		//To avoid blocking when checking for an exit or new connection, a select block is used
		select {
		case <-exit: //when handleExit() fills the exit channel
			//Create an exit message to send to each connected client
			exitMessage := Message{From: "Server", Content: "exit"}
			//Lock the connection map to ensure no new connections are added before this operation completes
			d.mu.Lock()

			//Iterate over all connections and send them a copy of the exit message
			for user, connection := range d.connectionMap {
				exitMessage.To = user

				binBuff := new(bytes.Buffer)

				gobObject := gob.NewEncoder(binBuff)
				gobObject.Encode(exitMessage)

				connection.Write(binBuff.Bytes())
			}
			//Unlock the map (not necessary but good practice)
			d.mu.Unlock()
			//The main thread returns and the program ends, killing all Goroutines it has previously spawned
			return

		case c := <-newConn: //When a new connection is accepted
			//Launches a dedicated Goroutine to handle communication with the client
			go handleConnection(c, d)

		}
	}
}
