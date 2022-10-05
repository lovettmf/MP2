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
	// key -> string
	// value -> connection
	d.connectionMap = make(map[string]net.Conn)

	return d, nil
}

//Add user to the connection map 
// key -> 
func (d *connections) Add(user string, c net.Conn) error {

	d.mu.Lock()
	defer d.mu.Unlock()
	d.connectionMap[user] = c
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

func handleConnection(c net.Conn, kill chan int, d *connections) {
	//https://gist.github.com/MilosSimic/ae7fe8d70866e89dbd6e84d86dc8d8d5

	//add initial message deciphering to get username
	netData, err := bufio.NewReader(c).ReadString('\n')
	fmt.Println(netData)
	if err != nil {
		fmt.Println(err)
		return
	}

	temp := strings.TrimSpace(string(netData))
	//fmt.Println(temp)
	// add connection to map
	// key -> username
	// value -> connection
	d.Add(temp, c)

	//we will see if this works

	tmp := make([]byte, 500)
	for {
		select {
		case <-kill:

			exitMessage := Message{To: "client", From: "server", Content: "exit"}
			bin_buf := new(bytes.Buffer)

			// create a encoder object
			gobobje := gob.NewEncoder(bin_buf)
			// encode buffer and marshal it into a gob object
			gobobje.Encode(exitMessage)

			c.Write(bin_buf.Bytes())

			break

		default:
			_, err := c.Read(tmp)
			if err != nil {
				fmt.Println(err)
				return
			}

			// convert bytes into Buffer (which implements io.Reader/io.Writer)
			tmpbuff := bytes.NewBuffer(tmp)
			// make a new Message
			tmpstruct := new(Message)

			// creates a decoder object
			gobobjdec := gob.NewDecoder(tmpbuff)
			// decodes buffer and unmarshals it into a Message struct
			gobobjdec.Decode(tmpstruct)

			dest, status := d.Lookup(tmpstruct.To)

			if status == "not found" {
				//send error message back to original client
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
	arguments := os.Args
	if len(arguments) == 1 {
		fmt.Println("Please provide a port number!")
		return
	}

	PORT := ":" + arguments[1]
	l, err := net.Listen("tcp", PORT)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer l.Close()

	//https://stackoverflow.com/questions/29948497/tcp-accept-and-go-concurrency-model

	exit := make(chan int, 1)
	kill := make(chan int, 1)
	newConn := make(chan net.Conn, 1)
	go handleExit(exit)

	d, err := New()

	count := 0
	for {

		go func(l net.Listener) {
			for {
				c, err := l.Accept()
				fmt.Println("Connecting...")
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
			if count == 0 {
				return
			}
			//Send kill signal for every connection handler to pass to client
			for i := 0; i < count; i++ {
				kill <- 1
			}
			//Kill server
			return

		case c := <-newConn:
			go handleConnection(c, kill, d)
			count++
		}
	}
}
