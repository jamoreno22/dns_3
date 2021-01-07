package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	lab3 "github.com/jamoreno22/dns_3/pkg/proto"
	"google.golang.org/grpc"
)

//DNSServer unimplemented
type DNSServer struct {
	lab3.UnimplementedDNSServer
}

var vectors []*lab3.VectorClock

var lg, _ = os.Create("log")
var wg sync.WaitGroup

func server() {
	defer wg.Done()
	// create a listener on TCP port 8000
	lis, err := net.Listen("tcp", "10.10.28.19:8000")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	// create a server instance
	dnss := DNSServer{}                          // create a gRPC server object
	grpcDNSServer := grpc.NewServer()            // attach the Ping service to the server
	lab3.RegisterDNSServer(grpcDNSServer, &dnss) // start the server

	log.Println("DNSServer_3 running ...")
	if err := grpcDNSServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %s", err)
	}
}

func spread() {
	defer wg.Done()
	for {
		var conn1 *grpc.ClientConn

		conn1, err1 := grpc.Dial("10.10.28.17", grpc.WithInsecure())
		if err1 != nil {
			log.Fatalf("did not connect: %s", err1)
		}

		dnsc1 := lab3.NewDNSClient(conn1)

		var conn2 *grpc.ClientConn

		conn2, err2 := grpc.Dial("10.10.28.18", grpc.WithInsecure())
		if err2 != nil {
			log.Fatalf("did not connect: %s", err2)
		}

		dnsc2 := lab3.NewDNSClient(conn2)
		for {
			time.Sleep(5 * time.Minute)
			logFile, _ := ioutil.ReadFile("lg")
			dnsc1.Spread(context.Background(), &lab3.Log{Name: "Log DNS3", Data: logFile})
			dnsc2.Spread(context.Background(), &lab3.Log{Name: "Log DNS3", Data: logFile})
		}
	}

}

func main() {
	runtime.GOMAXPROCS(2)

	wg.Add(2)

	go server()
	go spread()

	wg.Wait()
}

func isError(err error) bool {
	if err != nil {
		fmt.Println(err.Error())
	}

	return (err != nil)
}

// Action server side
func (s *DNSServer) Action(ctx context.Context, cmd *lab3.Command) (*lab3.VectorClock, error) {

	var registerLog, err3 = os.OpenFile("log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if isError(err3) {
		fmt.Printf("File opening error")

	}
	defer registerLog.Close()

	switch cmd.Action {
	case 1: //Create
		// check if file exists
		var _, err = os.Stat("ZF/" + cmd.Domain)

		// create file if not exists
		if os.IsNotExist(err) {
			var file, err1 = os.Create("ZF/" + cmd.Domain)
			vectors = append(vectors, &lab3.VectorClock{Name: cmd.Domain, Rv1: 0, Rv2: 0, Rv3: 0})
			if isError(err1) {
				fmt.Printf("File creation error")
			}
			defer file.Close()
		}

		var file, err2 = os.OpenFile("ZF/"+cmd.Domain, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if isError(err2) {
			fmt.Printf("File opening error")

		}
		defer file.Close()

		_, err = file.WriteString(cmd.Name + cmd.Domain + " IN A " + cmd.Ip + "\n")
		if isError(err) {
			fmt.Printf("File writing error")

		}
		_, err = registerLog.WriteString("Create " + cmd.Name + "." + cmd.Domain + "\n")
		if isError(err) {
			fmt.Printf("log writing error")
		}

	case 2: //Update
		input, err := ioutil.ReadFile("ZF/" + cmd.Domain)
		if err != nil {
			log.Fatalln(err)
		}

		lines := strings.Split(string(input), "\n")

		for i, line := range lines {
			if strings.Contains(line, cmd.Name) {
				if cmd.Option == "Name" {
					local := strings.Split(line, " ")
					localIP := local[len(local)-1]
					lines[i] = cmd.Parameter + cmd.Domain + " IN A " + localIP
				} else {
					lines[i] = cmd.Name + cmd.Domain + " IN A " + cmd.Parameter
				}
			}
		}
		output := strings.Join(lines, "\n")
		err = ioutil.WriteFile("ZF/"+cmd.Domain, []byte(output), 0644)
		if err != nil {
			log.Fatalln(err)
		}

		_, err = registerLog.WriteString("Update " + cmd.Option + " " + cmd.Parameter)
		if isError(err) {
			fmt.Printf("log writing error")
		}

	case 3: //Delete
		input, err := ioutil.ReadFile("ZF/" + cmd.Domain)
		if err != nil {
			log.Fatalln(err)
		}

		lines := strings.Split(string(input), "\n")
		deleted := false
		for i, line := range lines {
			if deleted == true {
				lines[i-1] = lines[i]
			}
			if strings.Contains(line, cmd.Name) {
				deleted = true
			}
		}
		lines = lines[:len(lines)-1]
		output := strings.Join(lines, "\n")
		err = ioutil.WriteFile("ZF/"+cmd.Domain, []byte(output), 0644)
		if err != nil {
			log.Fatalln(err)
		}

		_, err = registerLog.WriteString("Delete " + cmd.Name)
		if isError(err) {
			fmt.Printf("log writing error")
		}
	}

	for _, s := range vectors {
		if s.Name == cmd.Domain {
			s.Rv3++
			return s, nil
		}
	}
	return &lab3.VectorClock{}, nil
}

//Spread server side
func (s *DNSServer) Spread(ctx context.Context, lg *lab3.Log) (*lab3.Message, error) {

	return &lab3.Message{Text: "asdf"}, nil
}

//GetIP server side
func (s *DNSServer) GetIP(ctx context.Context, cmd *lab3.Command) (*lab3.PageInfo, error) {
	for _, s := range vectors {
		if s.Name == cmd.Domain {
			return &lab3.PageInfo{PageIp: cmd.Ip, Rv: s, DnsIP: "10.10.28.19:8000"}, nil
		}
	}
	return &lab3.PageInfo{}, nil
}
