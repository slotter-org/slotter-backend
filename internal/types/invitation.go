package types

import (
  "time"

  "gorm.io/gorm"
  "github.com/google/uuid"
)

type InvitationStatus string

const (
  InvitationStatusPending   InvitationStatus = "pending"
  InvitationStatusAccepted  InvitationStatus = "accepted"
  InvitationStatusCanceled  InvitationStatus = "canceled"
  InvitationStatusExpired   InvitationStatus = "expired"
)

type InvitationType string

const (
  InvitationTypeJoinWms                   InvitationType = "join_wms"
  InvitationTypeJoinWmsWithNewCompany     InvitationType = "join_wms_with_new_company"
  InvitationTypeJoinCompany               InvitationType = "join_company"
)

type Invitation struct {
  gorm.Model
  ID                  uuid.UUID                 `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
  InviteUserID        uuid.UUID                 `gorm:"index"`
  InviteUser          *User                     `gorm:"constraint:OnDelete:SET NULL;foreignKey:InviteUserID;references:ID"`
  WmsID               *uuid.UUID                 `gorm:"index"`
  Wms                 *Wms                      `gorm:"contraint:OnDelete:CASCADE;foreignKey:WmsID;references:ID"`
  CompanyID           *uuid.UUID                 `gorm:"index"`
  Company             *Company                  `gorm:"constraint:OnDelete:CASCADE;foreignKey:CompanyID;references:ID"`

  Token               string                    `gorm:"uniqueIndex;not null"`
  InvitationType      InvitationType            `gorm:"type:varchar(50);not null"`
  Status              InvitationStatus         `gorm:"type:varchar(50);not null;default:'pending'"`
  Email               *string                   `gorm:"column:email"`
  PhoneNumber         *string                   `gorm:"column:phone_number"`
  ExpiresAt           time.Time                 `gorm:"column:expires_at"`
  AvatarBucketKey     string                    `gorm:"column:avatar_bucket_key" json:"avatarBucketKey"`
  AvatarURL           string                    `gorm:"column:avatar_url" json:"avatarURL"`

  AcceptedAt          time.Time                 
  CanceledAt          time.Time

  CreatedAt           time.Time                 `gorm:"not null;default:now()"`
  UpdatedAt           time.Time                 `gorm:"not null;default:now()"`
}

func (Invitation) TableName() string {
  return "invitation"
}
