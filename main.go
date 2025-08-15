package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"
)

func main() {
	serverAddr := "127.0.0.1:9000" 
	deviceCount := 5               
	interval := 5 * time.Second    

	var wg sync.WaitGroup
	for i := 0; i < deviceCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			imei := randomIMEI()
			for {
				err := runDevice(serverAddr, imei, interval)
				if err != nil {
					log.Printf("[%s] disconnected: %v (reconnecting in 2s…)", imei, err)
					time.Sleep(2 * time.Second)
				}
			}
		}(i)
	}

	wg.Wait()
}

func runDevice(serverAddr, imei string, interval time.Duration) error {
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		return err
	}
	defer conn.Close()
	log.Printf("[%s] Connected to %s", imei, serverAddr)

	// Send IMEI
	imeiBytes := []byte(imei)
	if err := binary.Write(conn, binary.BigEndian, uint16(len(imeiBytes))); err != nil {
		return err
	}
	if _, err := conn.Write(imeiBytes); err != nil {
		return err
	}

	// Wait for IMEI ACK
	ack := make([]byte, 1)
	if _, err := conn.Read(ack); err != nil {
		return err
	}
	if ack[0] != 0x01 {
		return fmt.Errorf("invalid ACK: 0x%x", ack[0])
	}

	// Send AVL packets forever
	for {
		lat := 37.7749 + (rand.Float64()-0.5)/1000 // ± ~0.0005 deg
		lon := -122.4194 + (rand.Float64()-0.5)/1000
		packet := buildFakeAVLPacket(lat, lon)

		if _, err := conn.Write(packet); err != nil {
			return err
		}
		log.Printf("[%s] Sent AVL packet (lat=%.6f lon=%.6f)", imei, lat, lon)

		// Wait for record count ACK
		recAck := make([]byte, 4)
		if _, err := conn.Read(recAck); err != nil {
			return err
		}
		log.Printf("[%s] Received record ACK: %d", imei, binary.BigEndian.Uint32(recAck))

		time.Sleep(interval)
	}
}


func buildFakeAVLPacket(lat, lon float64) []byte {
	var buf bytes.Buffer
	
	buf.Write([]byte{0x00, 0x00, 0x00, 0x00})
	
	binary.Write(&buf, binary.BigEndian, uint32(0))

	
	buf.WriteByte(0x08)
	
	recordCount := byte(1)
	buf.WriteByte(recordCount)

	
	binary.Write(&buf, binary.BigEndian, uint64(time.Now().UnixMilli()))
	buf.WriteByte(0x00) 
	binary.Write(&buf, binary.BigEndian, int32(lon*10000000))
	binary.Write(&buf, binary.BigEndian, int32(lat*10000000))
	binary.Write(&buf, binary.BigEndian, int16(50))  
	binary.Write(&buf, binary.BigEndian, int16(180)) 
	buf.WriteByte(10)                                
	binary.Write(&buf, binary.BigEndian, int16(60))  

	
	buf.WriteByte(0x01) 
	buf.WriteByte(0x01) 
	buf.WriteByte(0x01) 
	buf.WriteByte(0x01) 
	buf.WriteByte(0x01) 

	
	buf.WriteByte(recordCount)

	
	crc := []byte{0x00, 0x00, 0x00, 0x00}
	packet := buf.Bytes()
	length := uint32(len(packet) - 8) 
	binary.BigEndian.PutUint32(packet[4:8], length)
	packet = append(packet, crc...)

	return packet
}

func randomIMEI() string {
	return fmt.Sprintf("3563070%08d", rand.Intn(100000000))
}



