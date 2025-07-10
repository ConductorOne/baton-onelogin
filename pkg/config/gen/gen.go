package main

import (
	cfg "github.com/conductorone/baton-onelogin/pkg/config"
	"github.com/conductorone/baton-sdk/pkg/config"
)

func main() {
	config.Generate("onelogin", cfg.Config)
}
