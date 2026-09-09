package activation

import "testing"

func TestValidateSectionItemsRequiresGoodsAndUsers(t *testing.T) {
	tests := []struct {
		name    string
		section string
		profile Profile
		wantErr bool
	}{
		{name: "goods empty", section: "goods", profile: Profile{}, wantErr: true},
		{name: "goods blank", section: "goods", profile: Profile{Goods: []string{" "}}, wantErr: true},
		{name: "goods valid", section: "goods", profile: Profile{Goods: []string{"Queijos"}}, wantErr: false},
		{name: "users empty", section: "users", profile: Profile{}, wantErr: true},
		{name: "users missing email", section: "users", profile: Profile{SystemUsers: []SystemUser{{Name: "Maria"}}}, wantErr: true},
		{name: "users valid", section: "users", profile: Profile{SystemUsers: []SystemUser{{Name: "Maria", Email: "maria@example.com"}}}, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSectionItems(tt.profile, tt.section)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateSectionItems() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
