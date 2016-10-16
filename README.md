# Connectors
A series of customized structs that conform to the net.Conn interface


[![Build Status](https://travis-ci.org/levigross/connectors.svg?branch=master)](https://travis-ci.org/levigross/connectors) 
[![GoDoc](https://godoc.org/github.com/levigross/connectors?status.svg)](https://godoc.org/github.com/levigross/connectors)
// [![Coverage Status](https://coveralls.io/repos/levigross/connectors/badge.svg)](https://coveralls.io/r/levigross/connectors)

LatencyConn
===========
This struct implements net.Conn but allows you to set the number of bytes you wish to read
within a given time quantum.
