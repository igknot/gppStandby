package remote

import (
	"bytes"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

func RemoteSsh(cmd string) (string, error) {
	//this is used when running in kubernetes using a secret base64
	sshfile := []byte(strings.Replace(os.Getenv("SSH_KEY"), "*", "\n", -1))

	//this is used when running local using env base64 ssh key
	//decoded, err := base64.StdEncoding.DecodeString(strings.Replace(os.Getenv("SSH_KEY"), "*", "\n", -1))
	//log.Print(decoded)
	//sshfile :=  []byte(decoded)


	signer, err := ssh.ParsePrivateKey(sshfile)
	if err != nil {
		log.Fatalf("unable to parse private key: %v", err)
	} else {
		log.Println("private key parsed")
	}

	config := &ssh.ClientConfig{
		User: sshUser(),

		Auth: []ssh.AuthMethod{

			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Connect
	addr := sshEndpoint()
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		log.Println("unable to connect via ssh to ", addr , err.Error())
		return "", err
	}
	// Create a session. It is one session per command.
	session, err := client.NewSession()
	if err != nil {
		log.Println("Unable to create ssh session", err.Error())
		return "", err
	}
	defer session.Close()
	var b bytes.Buffer  // import "bytes"
	session.Stdout = &b // get output

	err = session.Run(cmd)
	if err != nil {
		log.Println("Unable to execute command \n",cmd ,"\n", err.Error())

	}

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
