package utils

import (
	"strconv"

	"github.com/YoungGoofy/WebScanner/internal/services/scan"
)

func GetStatus(scanner scan.Scanner) (string, error) {
  // get status of passive scanner
  pscan := scanner.PassiveScanner
  pStatus, err := pscan.GetStatus()
  if err != nil {
    return "", err
  }
  intPStatus, err := strconv.Atoi(pStatus)
  if err != nil {
    return "", err
  }

  // get status of active scanner
  ascan := scanner.ActiveScanner
  aStatus, err := ascan.GetStatus()
  if err != nil {
    return "", err
  }
  intAStatus, err := strconv.Atoi(aStatus)
  if err != nil {
    return "", err
  }

  // get sum of full status
  result := strconv.Itoa((intPStatus + intAStatus * 17) / 18)
  return result, nil
}
