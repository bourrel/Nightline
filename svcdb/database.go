package svcdb

import (
	"fmt"

	bolt "github.com/johnnadratowski/golang-neo4j-bolt-driver"
)

var connPool bolt.DriverPool

// StartConnection the connection with neo4j
func StartDriver() bool {
	var err error
	//dbAddr := "bolt://neo4j:42-NightLineDB@37.187.124.34:8881/"
	dbAddr := "bolt://neo4j:42-NightLineDB@localhost:8881/"
	// dbAddr = "bolt://localhost:8881/"
	connMax := 50

	connPool, err = bolt.NewDriverPool(dbAddr, connMax)
	if err != nil {
		fmt.Println("StartConnection (NewDriverPool) : " + err.Error())
		return false
	}

	return true
}

// WaitConnection helper to claim a connection with retries & wait
// TODO
func WaitConnection(_ int) (bolt.Conn, error) {
	return ClaimConnection()
}

// ClaimConnection claim connection from pool
func ClaimConnection() (bolt.Conn, error) {
	conn, err := connPool.OpenPool()
	if err != nil {
		fmt.Println("ClaimConnection (OpenPool) : " + err.Error())
	}
	return conn, err
}

// CloseConnection close the connection
func CloseConnection(conn bolt.Conn) {
	err := conn.Close()
	if err != nil {
		fmt.Println("CloseConnection (reclaim) : " + err.Error())
	}
}

func CloseDriver() {
}
