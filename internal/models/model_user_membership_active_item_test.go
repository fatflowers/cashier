package models

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserMembershipActiveItem_TableName(t *testing.T) {
	var m UserMembershipActiveItem
	require.Equal(t, "user_membership_active_item", m.TableName())
}
