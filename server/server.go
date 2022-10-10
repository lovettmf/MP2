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

type Message struct {
	To      string
	From    string
	Content string
}

type connections struct {
	mu            sync.Mutex
	connectionMap map[string]net.Conn
}

// https://madflojo.medium.com/making-golang-packages-thread-safe-bb9592aeab92
func New() (*connections, error) {
	d := &connections{}
	// Create new map
	d.connectionMap = make(map[string]net.Conn)

	return d, nil
}

func (d *connections) Add(user string, c net.Conn) error {

	d.mu.Lock()
	defer d.mu.Unlock()
	d.connectionMap[user] = c
	return nil
}

func (d *connections) Delete(user string) error {

	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.connectionMap, user)
	return nil
}

func (d *connections) Lookup(user string) (net.Conn, string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	c, ok := d.connectionMap[user]
	if ok {
		return c, "valid"
	}
	return nil, "not found"
}

func handleConnection(c net.Conn, d *connections) {
	//https://gist.github.com/MilosSimic/ae7fe8d70866e89dbd6e84d86dc8d8d5

	//add initial message deciphering to get username
	tmp := make([]byte, 500)

	_, err := c.Read(tmp)
	if err != nil {
		fmt.Println(err)
		return
	}

	// convert bytes into Buffer (which implements io.Reader/io.Writer)
	tmpbuff := bytes.NewBuffer(tmp)
	tmpstruct := new(Message)

	// creates a decoder object
	gobobj := gob.NewDecoder(tmpbuff)
	// decodes buffer and unmarshalls it into a Message struct
	gobobj.Decode(tmpstruct)

	//Add the connection to the dictionary
	d.Add(tmpstruct.From, c)

	for {
		_, err = c.Read(tmp)
		if err != nil {
			fmt.Println(err)
			return
		}

		// convert bytes into Buffer (which implements io.Reader/io.Writer)
		tmpbuff = bytes.NewBuffer(tmp)
		tmpstruct = new(Message)

		// creates a decoder object
		gobobj = gob.NewDecoder(tmpbuff)
		// decodes buffer and unmarshals it into a Message struct
		gobobj.Decode(tmpstruct)

		//Lookup the connection object of the destination client
		dest, status := d.Lookup(tmpstruct.To)

		if status == "not found" {
			//send error message back to original client
			//this needs to be filled in and client needs it as well
			notFound := Message{To: tmpstruct.From, From: "Server", Content: tmpstruct.To + " not found\n"}
			bin_buf := new(bytes.Buffer)

			// create a encoder object
			gobobje := gob.NewEncoder(bin_buf)
			// encode buffer and marshal it into a gob object
			gobobje.Encode(notFound)

			c.Write(bin_buf.Bytes())
			continue
		}
		//remove the client from the map if it is exiting
		if tmpstruct.Content == "exiting" {
			d.Delete(tmpstruct.From)
			c.Close()
			return
		} else {
			//send to intended recipient
			bin_buf := new(bytes.Buffer)

			// create a encoder object
			gobobje := gob.NewEncoder(bin_buf)
			// encode buffer and marshal it into a gob object
			gobobje.Encode(tmpstruct)

			dest.Write(bin_buf.Bytes())
		}
	}

	c.Close()
}

func handleExit(exit chan int) {

	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print(">> ")
		text, _ := reader.ReadString('\n')
		input := strings.Split(text, "\n")

		if strings.ToLower(input[0]) == "exit" {
			exit <- 1
			return
		}
	}
}

func main() {

	//from linode
	//Perhaps check for valid port number
	arguments := os.Args
	if len(arguments) == 1 {
		fmt.Println("Please provide a port number!")
		return
	}

	//Start listening on given port
	PORT := ":" + arguments[1]
	l, err := net.Listen("tcp", PORT)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer l.Close()

	//https://stackoverflow.com/questions/29948497/tcp-accept-and-go-concurrency-model

	exit := make(chan int, 1)
	newConn := make(chan net.Conn, 1)
	go handleExit(exit)

	d, err := New() //new pointer to map, will hold username/connections

	for {

		go func(l net.Listener) {
			for {
				c, err := l.Accept()
				if err != nil {
					// handle error (and then for example indicate acceptor is down)
					newConn <- nil
					return
				}
				newConn <- c
			}
		}(l)

		select {
		case <-exit:
			//Needs to tell all clients to exit via a message. The main client should listen for this then, then return which kills all routines
			//send error message back to original client
			//this needs to be filled in and client needs it as well
			exitMessage := Message{From: "Server", Content: "exit"}
			d.mu.Lock()
			for user, connection := range d.connectionMap {
				exitMessage.To = user

				bin_buf := new(bytes.Buffer)

				// create a encoder object
				gobobje := gob.NewEncoder(bin_buf)
				// encode buffer and marshal it into a gob object
				gobobje.Encode(exitMessage)

				connection.Write(bin_buf.Bytes())
			}
			d.mu.Unlock()
			return

		case c := <-newConn:
			go handleConnection(c, d) //this needs to receive the message and update the map

		}
	}
}
