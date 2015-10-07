/*
 * Copyright(C) 2015 Simon Schmidt
 * 
 * This Source Code Form is subject to the terms of the
 * Mozilla Public License, v. 2.0. If a copy of the MPL
 * was not distributed with this file, You can obtain one at
 * http://mozilla.org/MPL/2.0/.
 */

package mailbottle

import "net/textproto"
//import "bufio"
import "io"
import "strings"
//import "fmt"
import "errors"

var TryAgain = errors.New("Try-Again-Later")
var PollEmpty = errors.New("Poll-Empty")

type Handler interface{
	HandleBottle(src io.Reader) (string,error)
	PollBottle() (bid string,f func(src io.Writer),e error)
	PurgeBottle(bid string) error
}

type Server struct{
	H Handler
	c *textproto.Conn
}
func (s *Server) Init(c io.ReadWriteCloser) {
	s.c = textproto.NewConn(c)
}
func (s *Server) message(n uint) {
	src := s.c.DotReader()
	r,e := s.H.HandleBottle(src)
	s.c.EndRequest(n)
	s.c.StartResponse(n)
	if e==nil {
		s.c.PrintfLine("%d %s",201,r)
	}else{
		enr := 501
		if e==TryAgain { enr = 301 }
		s.c.PrintfLine("%d %v",enr,e)
	}
	s.c.EndResponse(n)
}
func (s *Server) poll(n uint) {
	s.c.EndRequest(n)
	s.c.StartResponse(n)
	defer s.c.EndResponse(n)
	bid,f,e := s.H.PollBottle()
	if e==nil {
		s.c.PrintfLine("%d ID %s",200,bid)
		dw := s.c.DotWriter()
		defer dw.Close()
		f(dw)
	}else{
		enr := 501
		if e==PollEmpty { enr=401 }
		s.c.PrintfLine("%d %v",enr,e)
	}
}
func (s *Server) purge(n uint, bid string) {
	s.c.EndRequest(n)
	e := s.H.PurgeBottle(bid)
	s.c.StartResponse(n)
	if e==nil {
		s.c.PrintfLine("%d Purged %s",201,bid)
	}else{
		s.c.PrintfLine("%d %v",501,e)
	}
	s.c.EndResponse(n)
}

func (s *Server) ServeOneRequest() (err error){
	n := s.c.Next()
	s.c.StartRequest(n)
	line,e := s.c.ReadLine()
	if e!=nil { err = e }
	if err==nil {
		ls := strings.SplitN(line," ",2)
		cmd := ""
		if len(ls)>0 { cmd = ls[0] }
		switch cmd {
			case "MESSAGE":
			go s.message(n)
			case "POLL":
			go s.poll(n)
			case "PURGE":
			if len(ls)>=2 {
				go s.purge(n,ls[1])
			}
			default:
				s.c.EndRequest(n)
				s.c.StartResponse(n)
				s.c.PrintfLine("%d Error, unknown command",599)
				s.c.EndResponse(n)
		}
	}else{
		s.c.EndRequest(n)
		s.c.StartResponse(n)
		s.c.PrintfLine("%d Internal Server error",500)
		s.c.EndResponse(n)
	}
	return
}

