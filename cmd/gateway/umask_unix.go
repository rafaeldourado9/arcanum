//go:build unix

package main

import "syscall"

// Arquivos criados pelo gateway (banco de cada sessao, -wal, -shm, .json) nascem so para o
// dono (0600): o banco da sessao e a "senha" do numero de WhatsApp.
func init() { syscall.Umask(0o077) }
