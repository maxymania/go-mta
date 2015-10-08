/*
 * Copyright(C) 2015 Simon Schmidt
 * 
 * This Source Code Form is subject to the terms of the
 * Mozilla Public License, v. 2.0. If a copy of the MPL
 * was not distributed with this file, You can obtain one at
 * http://mozilla.org/MPL/2.0/.
 */

/*
 An SMTP server Implementation. The SMTP server is tightly coupled with the
 mailbottle package and makes use of the Mailbottle format.
*/
package smtpserv

import "fmt"

import "net"
import "net/textproto"
import "strings"
import "github.com/maxymania/go-mta/mailbottle"
import "io"
import "crypto/tls"

const (
	welcome = "220 service ready"
	action_ok = "250 OK"
	action_forward = "251 OK; forwarded"
	mail_input = "354 start mail input"
)

type ServerConfig struct{
	Handler mailbottle.Handler
	TlsConfig *tls.Config
}

type Server struct{
	*server
}
func (s *Server) Init(c *ServerConfig){
	s.server = new(server)
	s.backend = c.Handler
	s.tlsConfig = c.TlsConfig
	if s.tlsConfig!=nil {
		s.handlers["starttls"] = h_starttls
		s.exts = append(s.exts,"STARTTLS")
	}
	s.init()
}
func (s *Server) Serve(conn io.ReadWriteCloser) error {
	h := &handler{s:s.server,conn:textproto.NewConn(conn)}
	h.netconn,_ = conn.(net.Conn)
	h.startUp()
	for {
		e := h.serveOne()
		if e!=nil { return e }
	}
}

type cmdHandler func(h *handler,args []string,raw string) error

type errResp struct{
	code int
	msg string
}
func (e errResp) Error() string {
	return fmt.Sprintf("%d %s",e.code,e.msg)
}

type server struct{
	handlers map[string]cmdHandler
	exts []string
	backend mailbottle.Handler
	tlsConfig *tls.Config
}

func (s *server) init() {
	s.handlers = map[string]cmdHandler{
		"helo":h_helo,
		"ehlo":h_ehlo,
		"mail":h_mail,
		"rcpt":h_rcpt,
		"data":h_data,
		"quit":h_quit,
	}
	s.exts = make([]string,0,32)
	/* This simply means, that the emails may be 8 Bit (utf-8, yenc, etc.) */
	s.exts = append(s.exts,"8BITMIME")
}

type handler struct{
	s *server
	conn *textproto.Conn
	netconn net.Conn
	clientString string
	mbottle mailbottle.BottleInfo
}

func (h *handler) startUp() {
	h.conn.PrintfLine(welcome)
}

func (h *handler) serveOne() error {
	l,e := h.conn.ReadLine()
	if e!=nil { return e }
	arr := strings.Split(l," ")
	if len(arr)==0 {
		h.conn.PrintfLine("500 Syntax error, command unrecognized")
		return nil
	}
	f,ok := h.s.handlers[strings.ToLower(arr[0])]
	if !ok {
		h.conn.PrintfLine("502 Command not implemented")
		return nil
	}
	err := f(h,arr,l)
	if e,ok := err.(errResp); ok {
		h.conn.PrintfLine("%d %s",e.code,e.msg)
		return nil
	}
	return err
}

func h_helo(h *handler,args []string,raw string) error {
	if len(args)<2 { return errResp{501,"parameter"} }
	h.clientString = args[1]
	h.conn.PrintfLine(action_ok)
	return nil
}
func h_ehlo(h *handler,args []string,raw string) error {
	if len(args)<2 { return errResp{501,"parameter"} }
	h.clientString = args[1]
	message := "OK"
	for _,ext := range h.s.exts {
		h.conn.PrintfLine("250-%s",message)
		message = ext
	}
	h.conn.PrintfLine("250 %s",message)
	return nil
}

func h_mail(h *handler,args []string,raw string) error {
	if len(args)<2 { return errResp{501,"parameter"} }
	i := strings.Index(raw,"<")
	if i<0 { return errResp{501, "invalid parameter"} }
	raw = raw[i+1:]
	i = strings.Index(raw,">")
	raw = raw[:i]
	h.mbottle.From = append(h.mbottle.From,raw)
	h.conn.PrintfLine(action_ok)
	return nil
}

func h_rcpt(h *handler,args []string,raw string) error {
	if len(args)<2 { return errResp{501,"parameter"} }
	i := strings.Index(raw,"<")
	if i<0 { return errResp{501, "invalid parameter"} }
	raw = raw[i+1:]
	i = strings.Index(raw,">")
	raw = raw[:i]
	h.mbottle.RcptTo = append(h.mbottle.RcptTo,raw)
	h.conn.PrintfLine(action_ok)
	return nil
}

func h_data(h *handler,args []string,raw string) error {
	if len(h.mbottle.From)==0 || len(h.mbottle.RcptTo)==0 {
		return errResp{503,"Bad sequence of commands"}
	}
	h.conn.PrintfLine(mail_input)
	w,r := mailbottle.NewPipe()
	e := make(chan error,1)
	go func() {
		_,ee := h.s.backend.HandleBottle(r)
		e <- ee
	}()
	iow := w.WriteData(&h.mbottle)
	h.mbottle = mailbottle.BottleInfo{}
	io.Copy(iow,h.conn.DotReader())
	iow.Close()
	ee := <- e
	if ee!=nil {
		h.conn.PrintfLine("554 Transaction failed")
	}
	h.conn.PrintfLine(action_ok)
	return nil
}

func h_quit(h *handler,args []string,raw string) error {
	h.conn.PrintfLine("221 closing channel")
	h.conn.Close()
	return io.EOF
}

func h_starttls(h *handler,args []string,raw string) error {
	if h.s.tlsConfig==nil { return errResp{500,"unknown"} }
	if h.netconn==nil { return errResp{500,"unknown"} }
	h.conn.PrintfLine("220 go ahead")
	cc := tls.Server(h.netconn,h.s.tlsConfig)
	e := cc.Handshake()
	if e!=nil { return e }
	*h = handler{s:h.s,conn:textproto.NewConn(cc),netconn:cc}
	return nil
}


