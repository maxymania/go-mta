/*
 * Copyright(C) 2015 Simon Schmidt
 * 
 * This Source Code Form is subject to the terms of the
 * Mozilla Public License, v. 2.0. If a copy of the MPL
 * was not distributed with this file, You can obtain one at
 * http://mozilla.org/MPL/2.0/.
 */

package mailbottle

//import "net/textproto"
import "bufio"
import "bytes"
import "strings"
import "fmt"
import "io"

type BottleInfo struct{
	From   []string
	RcptTo []string
	MIME8B bool // BODY=8BITMIME
}

type Reader struct{
	*bufio.Reader
}

func (r *Reader) line() (string,error) {
	l,p,e := r.ReadLine()
	if e!=nil { return "",e }
	rdr := bytes.NewBuffer(l)
	for p {
		l,p,e = r.ReadLine()
		if e!=nil { p = false }
		rdr.Write(l)
	}
	return rdr.String(),e
}

func (r *Reader) ReadData(b *BottleInfo) (io.Reader,error) {
	for {
		l,e := r.line()
		if e!=nil { return nil,e }
		switch {
		case strings.HasPrefix(l,"BODY-8BITMIME"):
			b.MIME8B = true
		case strings.HasPrefix(l,"FROM:"):
			b.From = append(b.From,l[5:])
		case strings.HasPrefix(l,"RCPT-TO:"):
			b.RcptTo = append(b.RcptTo,l[8:])
		case strings.HasPrefix(l,"DATA"):
			return r,nil
		}
	}
	return nil,nil
}

func NewReader(r io.Reader) *Reader {
	return &Reader{bufio.NewReader(r)}
}



type Writer struct{
	*bufio.Writer
	io.Closer
}

/* FROM equivalent */
func (w *Writer) From(s string) {
	fmt.Fprintf(w,"FROM:%s\n",s)
}

/* RCPT TO equivalent */
func (w *Writer) RcptTo(s string) {
	fmt.Fprintf(w,"RCPT-TO:%s\n",s)
}

/* Begins the Data part of the Bottle. */
func (w *Writer) Data() io.WriteCloser {
	fmt.Fprintf(w,"DATA\n")
	return w
}

/*
 Sends From and RcptTo pieces of a BottleInfo object.
 In Follow-up it starts the data section.
*/
func (w *Writer) WriteData(b *BottleInfo) io.WriteCloser {
	if b.MIME8B {
		fmt.Fprintf(w,"BODY-8BITMIME\n")
	}
	for _,s := range b.From {
		w.From(s)
	}
	for _,s := range b.RcptTo {
		w.RcptTo(s)
	}
	return w.Data()
}

/* This function should not be used directly. See .Data(). */
func (w *Writer) Close() error {
	w.Flush()
	return w.Closer.Close()
}

func NewPipe() (*Writer,io.Reader) {
	r,w := io.Pipe()
	return &Writer{bufio.NewWriter(w),w},r
}


