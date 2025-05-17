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
  InvitationStatusRejected  InvitationStatus = "rejected"
)

type InvitationType string

const (
  InvitationTypeJoinWms                   InvitationType = "join_wms"
  InvitationTypeJoinWmsWithNewCompany     InvitationType = "join_wms_with_new_company"
  InvitationTypeJoinCompany               InvitationType = "join_company"
)

type Invitation struct {
  gorm.Model
  ID                  uuid.UUID                 `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
  InviteUserID        uuid.UUID                 `gorm:"index" json:"user_id"`
  InviteUser          *User                     `gorm:"constraint:OnDelete:SET NULL;foreignKey:InviteUserID;references:ID"`
  WmsID               *uuid.UUID                `gorm:"index" json:"wms_id,omitempty"`
  Wms                 *Wms                      `gorm:"contraint:OnDelete:CASCADE;foreignKey:WmsID;references:ID"`
  CompanyID           *uuid.UUID                `gorm:"index" json:"company_id,omitempty"`
  Company             *Company                  `gorm:"constraint:OnDelete:CASCADE;foreignKey:CompanyID;references:ID"`

  Name                *string                   `gorm:"column:name" json:"name,omitempty"`
  RoleID              *uuid.UUID                `gorm:"column:role_id" json:"role_id,omitempty"`

  Token               string                    `gorm:"uniqueIndex;not null" json:"token"`
  InvitationType      InvitationType            `gorm:"type:varchar(50);not null" json:"invitation_type,omitempty"`
  Status              InvitationStatus          `gorm:"type:varchar(50);not null;default:'pending'" json:"status"`
  Message             *string                   `gorm:"column:message" json:"message,omitempty"`
  Email               *string                   `gorm:"column:email" json:"email,omitempty"`
  PhoneNumber         *string                   `gorm:"column:phone_number" json:"phone_number,omitempty"`
  ExpiresAt           time.Time                 `gorm:"column:expires_at" json:"expires_at"`
  AvatarBucketKey     string                    `gorm:"column:avatar_bucket_key" json:"avatarBucketKey"`
  AvatarURL           string                    `gorm:"column:avatar_url" json:"avatarURL"`

  AcceptedAt          *time.Time                 `gorm:"column:accepted_at" json:"accepted_at,omitempty"`
  RejectedAt          *time.Time                 `gorm:"column:rejected_at" json:"rejected_at,omitempty"`
  ExpiredAt           *time.Time                 `gorm:"column:expired_at" json:"expired_at,omitempty"`
  CanceledAt          *time.Time                 `gorm:"column:canceled_at" json:"canceled_at,omitempty"`


  CreatedAt           time.Time                 `gorm:"not null;default:now()" json:"created_at,omitempty"`
  UpdatedAt           time.Time                 `gorm:"not null;default:now()" json:"updated_at,omitempty"`
}

func (Invitation) TableName() string {
  return "invitation"
}
