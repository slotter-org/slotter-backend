package utils

import (
  "context"
  "fmt"

  "golang.org/x/crypto/bcrypt"

  "github.com/yungbote/slotter/backend/internal/normalization"
  "github.com/yungbote/slotter/backend/internal/logger"
  "github.com/yungbote/slotter/backend/internal/types"
  "github.com/yungbote/slotter/backend/internal/repos"
)

func InputValidation (ctx context.Context, ffor string, userRepo repos.UserRepo, log  *logger.Logger, user *types.User, newCompanyName, newWmsName string) error {
  validatedFor := normalization.ParseInputString(ffor)
  if validatedFor == "" {
    log.Warn("for string is nil, needs to be either 'registration' or 'login'. Returning error", "for", validatedFor)
    return fmt.Errorf("for string is nil, needs to be either 'registration' or 'login': '%s'", validatedFor)
  }
  switch validatedFor {
  case "registration":
    if err := handleRegisterInputValidation(ctx, userRepo, log, user); err != nil {
      return err
    }
  case "login":
    if err := handleLoginInputValidation(ctx, log, newWmsName, newCompanyName); err != nil {
      return err
    }
  default:
    log.Warn("for string is invalid, needs to be either 'registration' or 'login'. Returning error", "for", validatedFor)
    return fmt.Errorf("for string is invalid, needs to be either 'registration' or 'login': '%s'", validatedFor)
  }
  return nil
}

func handleRegisterInputValidation(ctx context.Context, userRepo repos.UserRepo, log *logger.Logger, user *types.User) error {
  //1) Check if user is empty
  if user == nil {
    log.Warn("User is nil, cannot proceed further. Returning error", "user", user)
    return fmt.Errorf("No user given, cannot proceed any further.")
  }

  //2) Check Email
  if user.Email == "" {
    log.Warn("Email is nil, cannot proceed further. Returning error", "email", user.Email)
    return fmt.Errorf("an email is required to register.")
  }
  emailExists, err := userRepo.EmailExists(ctx, nil, user.Email)
  if err != nil {
    log.Warn("Failed to check if user email exists, error from UserRepo. Returning an error.", "error", err)
    return fmt.Errorf("Failed checking user email '%s' existence: %w", user.Email, err)
  }
  if emailExists {
    log.Warn("Email is already in use, cannot continue. Returning an error.", "emailExists", emailExists)
    return fmt.Errorf("email is already in use.")
  }

  //3) Check Phone Number
  if *user.PhoneNumber != "" {
    phoneExists, err := userRepo.PhoneNumberExists(ctx, nil, *user.PhoneNumber)
    if err != nil {
      log.Warn("Failed to check if user phone number exists, error from UserRepo. Returning an error.", "error", err)
      return fmt.Errorf("Failed checking user phone number '%s' existence: %w", user.PhoneNumber, err)
    }
    if phoneExists {
      log.Warn("Phone Number is already in use, cannot continue. Returning an error.", "phoneExists", phoneExists)
      return fmt.Errorf("phone number is already in use.")
    }
  }

  //4) Check Password
  if user.Password == "" {
    log.Warn("Password is nil, cannot proceed further. Returning error", "password", user.Password)
    return fmt.Errorf("a password is required to register.")
  }

  //4) Check FirstName
  if user.FirstName == "" {
    log.Warn("First Name is nil, cannot proceed further. Returning error", "firstName", user.FirstName)
    return fmt.Errorf("a first name is required to register.")
  }

  //5) Check LastName
  if user.LastName == "" {
    log.Warn("Last Name is nil, cannot proceed further. Returning error", "lastName", user.LastName)
    return fmt.Errorf("a last name is required to register.")
  }

  //6) Check UserType
  if user.UserType == "" {
    log.Warn("User Type is nil, cannot proceed further. Returning error", "userType", user.UserType)
    return fmt.Errorf("user type is required to register.")
  } else if user.UserType != "wms" && user.UserType != "company" {
    log.Warn("User Type must be either 'wms' or 'company' to proceed further. Returning error", "userType", user.UserType)
    return fmt.Errorf("user type is set incorrectly (must be either 'wms' or 'company'): '%s'", user.UserType)
  } else {
  }
  return nil
}

func handleLoginInputValidation(ctx context.Context, log *logger.Logger, email, password string) error {
  //1) Check Email
  if email == "" {
    log.Warn("Email is an empty string, Cannot proceed.", "email", email)
    return fmt.Errorf("Email is an empty string, Cannot proceed.")
  }
  
  //2) Check Password
  if password == "" {
    log.Warn("Password is an empty string, Cannot proceed.", "password", password)
    return fmt.Errorf("Password is an empty string, Cannot proceed.")
  }
  return nil
}

func HashPassword(ctx context.Context, log *logger.Logger, user *types.User) error {
  hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
  if err != nil {
    log.Warn("Failure to hash password for user. Returning error", "password", user.Password)
    return fmt.Errorf("Failed to hash password for user.")
  }
  user.Password = string(hashedPassword)
  return nil
}

func NormalizeUserFields(ctx context.Context, user *types.User) {
  user.UserType = normalization.ParseInputString(user.UserType)
  user.Email = normalization.ParseInputString(user.Email)
  user.PhoneNumber = normalization.ParseInputStringPtr(user.PhoneNumber)
  user.Password = normalization.ParseInputString(user.Password)
  user.FirstName = normalization.ParseInputString(user.FirstName)
  user.LastName = normalization.ParseInputString(user.LastName)
}



