package templates

import (
	"bytes"
	"html/template"
)

type InvitationType string

const (
	InvitationTypeJoinWms									InvitationType = "join_wms"
	InvitationTypeJoinCompany							InvitationType = "join_company"
	InvitationTypeJoinWmsWithNewCompany		InvitationType = "join_wms_with_new_company"
)

type InvitationEmailData struct {
	Logo						string
	InvitationLink 	string
	AvatarURL 			string
	InvitationType 	InvitationType
	WmsName 				string
	CompanyName			string
}

const invitationHTML = `
<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8"/>
  <title>Slotter Invitation</title>
  <style>
    body {
      margin: 0;
      padding: 0;
      font-family: Arial, sans-serif;
      background-color: #f5f5f5;
      color: #333;
    }
    .email-container {
      width: 100%;
      max-width: 600px;
      margin: 0 auto;
      background-color: #ffffff;
      border-radius: 6px;
      overflow: hidden;
      box-shadow: 0 2px 5px rgba(0,0,0,0.1);
    }
    .header {
      background-color: #333;
      padding: 20px;
      text-align: center;
      color: #fff;
    }
    .header img {
      width: 120px; /* Adjust your brand logo size */
      height: auto;
      margin-bottom: 10px;
    }
    .header h1 {
      margin: 10px 0 0;
      font-size: 24px;
    }
    .content {
      padding: 20px;
      text-align: left;
    }
    .avatar-container {
      text-align: center;
      margin: 10px 0 20px;
    }
    .avatar-container img {
      width: 60px;
      height: 60px;
      border-radius: 50%;
    }
    .button-container {
      text-align: center;
      margin: 20px 0;
    }
    .cta-button {
      display: inline-block;
      padding: 12px 24px;
      background-color: #333;
      color: #ffffff;
      text-decoration: none;
      border-radius: 4px;
      font-weight: bold;
    }
    .footer {
      font-size: 12px;
      color: #999;
      text-align: center;
      padding: 10px 20px;
    }
    .highlight {
      font-weight: bold;
      color: #333;
    }
  </style>
</head>
<body>
  <table class="email-container" role="presentation" cellspacing="0" cellpadding="0">
    <tr>
      <td>
        <!-- HEADER SECTION -->
        <div class="header">
          <!-- Our brand or Slotter logo -->
          <img src="{{.Logo}}" alt="Slotter Brand Logo" />
          <h1>Welcome to Slotter!</h1>
        </div>

        <!-- BODY CONTENT -->
        <div class="content">
          {{if .RecipientName}}
            <p>Hi <span class="highlight">{{.RecipientName}}</span>,</p>
          {{else}}
            <p>Hello,</p>
          {{end}}

          <!-- Show the WMS/Company avatar in the body -->
          <div class="avatar-container">
            <img src="{{.AvatarURL}}" alt="Organization Avatar" />
          </div>

          {{if eq .InvitationType "join_wms"}}
            <p>You’re invited to register as a user with 
               <span class="highlight">{{.WmsName}}</span>.</p>
          {{end}}

          {{if eq .InvitationType "join_company"}}
            <p>You’re invited to register as a user with 
               <span class="highlight">{{.CompanyName}}</span>.</p>
          {{end}}

          {{if eq .InvitationType "join_wms_with_new_company"}}
            <p>You’re invited to register a new company under 
               <span class="highlight">{{.WmsName}}</span>.</p>
          {{end}}

          <p>We're excited to have you on board! Please click 
             the button below to accept your invitation and set up your account.</p>

          <div class="button-container">
            <a class="cta-button" href="{{.InvitationLink}}">Accept Invitation</a>
          </div>
        </div>

        <!-- FOOTER SECTION -->
        <div class="footer">
          <p>&copy; 2025 Slotter Inc. All rights reserved.</p>
        </div>
      </td>
    </tr>
  </table>
</body>
</html>
`

func RenderInvitationHTML(data InvitationEmailData) (string, error) {
	tmpl, err := template.New("invitation").Parse(invitationHTML)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
