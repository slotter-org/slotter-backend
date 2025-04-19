package types

import (
  "time"

  "gorm.io/gorm"
  "github.com/google/uuid"
)

type User struct {
  gorm.Model
  ID                  uuid.UUID                 `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
  UserType            string                    `gorm:"column:user_type" json:"userType"`
  WmsID               *uuid.UUID                `gorm:"index" json:"wmsID,omitempty"`
  Wms                 *Wms                      `gorm:"constraint:OnDelete:CASCADE;foreignKey:WmsID;references:ID" json:"wms,omitempty"`
  CompanyID           *uuid.UUID                `gorm:"index" json:"companyID,omitempty"`
  Company             *Company                  `gorm:"constraint:OnDelete:CASCADE;foreignKey:CompanyID;references:ID" json:"company,omitempty"`
  RoleID              *uuid.UUID                `gorm:"index" json:"roleID,omitempty"`
  Role                *Role                     `gorm:"constraint:OnDelete:SET NULL;foreignKey:RoleID;references:ID" json:"role,omitempty"`

  Email               string                    `gorm:"uniqueIndex;not null;column:email" json:"email"`
  PhoneNumber         *string                   `gorm:"column:phone_number" json:"phoneNumber,omitempty"`
  Password            string                    `gorm:"not null;column:password" json:"-"`
  FirstName           string                    `gorm:"not null;column:first_name" json:"firstName"`
  LastName            string                    `gorm:"not null;column:last_name" json:"lastName"`
  AvatarBucketKey     string                    `gorm:"column:avatar_bucket_key" json:"avatarBucketKey"`
  AvatarURL           string                    `gorm:"column:avatar_url" json:"avatarURL"`

  CreatedAt           time.Time                 `gorm:"not null;default:now()" json:"createdAt"`
  UpdatedAt           time.Time                 `gorm:"not null;default:now()" json:"updatedAt"`
}

func (User) TableName() string {
  return "user"
}
