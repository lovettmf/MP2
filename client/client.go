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

type Message struct {
	To      string
	From    string
	Content string
}

func rec(c net.Conn, exit chan int) {
	for {
		// create a temp buffer
		tmp := make([]byte, 500)
		c.Read(tmp)

		// convert bytes into Buffer (which implements io.Reader/io.Writer)
		tmpbuff := bytes.NewBuffer(tmp)
		tmpstruct := new(Message)

		// creates a decoder object
		gobobjdec := gob.NewDecoder(tmpbuff)
		// decodes buffer and unmarshals it into a Message struct
		gobobjdec.Decode(tmpstruct)

		if tmpstruct.Content == "exit" {
			exit <- 1
			return
		}

		fmt.Print("From " + tmpstruct.From + ": " + tmpstruct.Content + ">> ")
	}
}

func send(c net.Conn, exit chan int, username string, freshInput chan int) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(">> ")
	text, _ := reader.ReadString('\n')

	input := strings.Split(text, " ")

	if strings.ToLower(input[0]) == "send" {

		to := input[1]                          //username of recipient
		content := strings.Join(input[2:], " ") //message content

		message := Message{To: to, From: username, Content: content}

		bin_buf := new(bytes.Buffer)

		// create a encoder object
		gobobj := gob.NewEncoder(bin_buf)
		// encode buffer and marshal it into a gob object
		gobobj.Encode(message)

		c.Write(bin_buf.Bytes())

	} else if strings.ToLower(strings.TrimSpace(input[0])) == "exit" { //Check for exit command
		exit <- 1
		return
	}
	freshInput <- 1
}

func main() {
	arguments := os.Args
	if len(arguments) < 4 {
		fmt.Println("Please provide host, port, and username.")
		return
	}

	username := arguments[3]

	CONNECT := arguments[1] + ":" + arguments[2]
	c, err := net.Dial("tcp", CONNECT)
	if err != nil {
		fmt.Println(err)
		return
	}
	//Initial connection message so server can document username
	connectMessage := Message{From: username, Content: "initial message"}
	bin_buf := new(bytes.Buffer)

	// create a encoder object
	gobobje := gob.NewEncoder(bin_buf)
	// encode buffer and marshal it into a gob object
	gobobje.Encode(connectMessage)

	c.Write(bin_buf.Bytes())

	exit := make(chan int, 1)
	freshInput := make(chan int, 1)

	go rec(c, exit)

	for {
		go send(c, exit, username, freshInput)

		select {
		case <-freshInput:
			continue
		case <-exit:
			//Initial connection message so server can document username
			exitMessage := Message{From: username, Content: "exiting"}
			bin_buf := new(bytes.Buffer)

			// create a encoder object
			gobobje := gob.NewEncoder(bin_buf)
			// encode buffer and marshal it into a gob object
			gobobje.Encode(exitMessage)

			c.Write(bin_buf.Bytes())
			fmt.Println("connection closed")
			exit <- 1
			return
		}

	}

}
