package main

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"strings"
)

// Message contains sender, destination, and content information to be sent between client and server
type Message struct {
	To      string
	From    string
	Content string
}

//This function is run in a Goroutine and endlessly reads new messages from the server 
func rec(c net.Conn, exit chan int) {

	for {
		//An array of bytes to be used later as a buffer to send/receive Message objects
		temp := make([]byte, 500)
		//Read from net.Conn object into the array of bytes
		c.Read(temp)

		//The array of bytes received becomes a buffer that can be encoded
		tempBuff := bytes.NewBuffer(temp)
		//An empty Message struct for the incoming message to be dumped into
		tempMsg := new(Message)

		//Creates a new gob Decoder object
		gobObj := gob.NewDecoder(tempBuff)
		//Decodes the gob stream that was read into the buffer and offloads it into the empty Message struct
		gobObj.Decode(tempMsg)

		//If the server is exited alert the main thread and return
		if tempMsg.Content == "exit" {
			exit <- 1
			return
		}

		//Print the received message to the standard output
		fmt.Print("From " + tempMsg.From + ": " + tempMsg.Content + ">> ")
	}
}

func send(c net.Conn, exit chan int, username string, freshInput chan int) {

	//This function sends messages from the client to the server to be forwarded to its intended destination 

	//Read input from the user
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(">> ")
	text, _ := reader.ReadString('\n')

	input := strings.Split(text, " ")

	//Check to see if the input is a message to be sent
	//Note that if there is invalid input this function will return and the program will continue as before
	if strings.ToLower(input[0]) == "send" {

		//extract the username of the intended recipient
		//If the username is invalid, the server will alert the client
		to := input[1]
		//Extract the content of the message
		content := strings.Join(input[2:], " ") //message content

		//Create a Message object with this information
		message := Message{To: to, From: username, Content: content}

		//Create a new byte buffer
		binBuff := new(bytes.Buffer)

		//Creates a new gob Encoder object using the byte buffer
		gobObject := gob.NewEncoder(binBuff)
		//Encodes the message into a gob stream then leads it into the byte buffer
		gobObject.Encode(message)

		//The encoded byte buffer is written over the TCP connection to the server
		c.Write(binBuff.Bytes())

	} else if strings.ToLower(strings.TrimSpace(input[0])) == "exit" { //Check for exit command
		//Alert the main routine via the exit channel and return
		exit <- 1
		return
	}
	//Alerts the main routine to run this function in another Goroutine and accept further input
	freshInput <- 1
}

func main() {

	//Reads the command line arguments for a host address and port to connect to, as well as a username for the client
	arguments := os.Args
	if len(arguments) < 4 {
		fmt.Println("Please provide host, port, and username.")
		return
	}

	//Extract username
	username := arguments[3]

	//Begin the TCP connection to the server
	CONNECT := arguments[1] + ":" + arguments[2]
	c, err := net.Dial("tcp", CONNECT)
	if err != nil {
		fmt.Println(err)
		return
	}

	//Send an initial connection message so server can document username
	connectMessage := Message{From: username, Content: "initial message"}
	binBuff := new(bytes.Buffer)

	gobObject := gob.NewEncoder(binBuff)
	gobObject.Encode(connectMessage)

	//send message
	c.Write(binBuff.Bytes())

	//Channel to indicate exit input or to relaunch the send function
	exit := make(chan int, 1)
	freshInput := make(chan int, 1)

	//Launch a Goroutine to listen for messages
	go rec(c, exit)

	for {

		//Launch a routine to accept input
		go send(c, exit, username, freshInput)

		//Will block until it needs to accept new input or until an exit command is inputted
		select {
		case <-freshInput:
			continue
		case <-exit:
			//This will send a message to the server so it knows this client is exiting
			exitMessage := Message{From: username, Content: "exiting"}
			binBuff := new(bytes.Buffer)

			gobObj := gob.NewEncoder(binBuff)
			gobObj.Encode(exitMessage)

			c.Write(binBuff.Bytes())
			fmt.Println("connection closed") //Alert the user that the connection has been closed
			exit <- 1
			return //return, killing any Goroutines spawned by this main thread
		}
	}
}
