package templates

import (
    "testing"

    "github.com/jefersonMarques/apresentacao-viagate/internal/activation"
)

func TestActivationAdminStatusLabel(t *testing.T) {
    tests := []struct {
        name string
        item activation.AdminItem
        expected string
    }{
        {
            name: "customer submitted without commercial operation",
            item: activation.AdminItem{Status: "completed", OperationReady: false},
            expected: "Aguardando operação comercial",
        },
        {
            name: "customer submitted and commercial operation complete",
            item: activation.AdminItem{Status: "completed", OperationReady: true},
            expected: "Dados recebidos",
        },
        {
            name: "internal setup in progress",
            item: activation.AdminItem{Status: "under_internal_setup", OperationReady: true},
            expected: "Em implantação",
        },
    }

    for _, test := range tests {
        t.Run(test.name, func(t *testing.T) {
            actual := ActivationAdminStatusLabel(test.item)
            if actual != test.expected {
                t.Errorf("status = %q; expected %q",actual,test.expected)
            }
        })
    }
}

func TestActivationOperationCompleteRejectsInvalidDates(t *testing.T) {
    profile := activation.Profile{
        OperationType: "normal",
        Insurer: "Insurer",
        PolicyStartDate: "2027-11-16",
        PolicyEndDate: "2026-11-16",
    }
    if ActivationOperationComplete(profile) {
        t.Fatal("reversed insurance validity must not be ready")
    }

    profile.PolicyEndDate = "2028-11-16"
    if !ActivationOperationComplete(profile) {
        t.Fatal("valid commercial operation should be ready")
    }
}
