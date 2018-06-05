package remote

import (
	"os"
	"golang.org/x/crypto/ssh"
	"bytes"
	"io/ioutil"
	"fmt"
)

func RemoteSsh(cmd string) (string, error) {
	sshfile := os.Getenv("SSH_KEY")
	config := &ssh.ClientConfig{
		User: sshUser(),

		Auth: []ssh.AuthMethod{
			PublicKeyFile(sshfile)},
	}

	config.HostKeyCallback = ssh.InsecureIgnoreHostKey()

	// Connect
	addr := sshEndpoint()
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {

		return "", err
	}
	// Create a session. It is one session per command.
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	var b bytes.Buffer  // import "bytes"
	session.Stdout = &b // get output

	err = session.Run(cmd)

	return b.String(), err
}

func sshUser() string {
	return os.Getenv("SSH_USER")
}

func sshEndpoint() string {
	return os.Getenv("SSH_ENDPOINT")
}

func PublicKeyFile(file string) ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		fmt.Println("Parsing failed", err.Error())
		return nil
	}

	return ssh.PublicKeys(key)
}
