package main

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"time"
)

func main() {
	err := connectToHost("anything", "localhost:2200")
	if err != nil {
		fmt.Println("ERR:", err)
	}
}

func PublicKeyFile() (ssh.AuthMethod, error) {
	buffer, err := ioutil.ReadFile("./out")
	if err != nil {
		return nil, err
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil, err
	}
	return ssh.PublicKeys(key), nil
}

func connectToHost(user, host string) error {
	sshConfig := &ssh.ClientConfig{
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	client, err := ssh.Dial("tcp", host, sshConfig)
	if err != nil {
		return err
	}

	_, uuid, err := client.SendRequest("hackme", true, nil)
	if err != nil {
		return err
	}
	fmt.Printf("UUID is %q\n", string(uuid))

	server, requests, err := client.OpenChannel("hackme", nil)
	if err != nil {
		return err
	}
	go makeChatRobot(server)
	return makeBashServer(server, requests)
}

func makeChatRobot(server ssh.Channel) error {
	go func() {
		for {
			<-time.After(time.Second)
			server.SendRequest("chat", false, []byte(fmt.Sprintf("Server Time is : %s", time.Now())))
		}
	}()
	return nil
}
