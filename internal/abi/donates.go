package abi

import (
	_ "embed"
)

// DonatesABI содержит ABI смарт-контракта для донатов, загруженное из файла
//
//go:embed Donates.abi
var DonatesABI string
