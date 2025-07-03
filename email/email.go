package email

import (
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"time"

	"gopkg.in/gomail.v2"
)

// Config contiene la configuración del servidor SMTP
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// SendMagicLink envía un correo con un enlace mágico usando Amazon SES
func SendMagicLink(email, token, host string) error {
	// Configuración específica para Amazon SES
	config := Config{
		Host:     os.Getenv("SMTP_HOST"),     // Usar SMTP_HOST del entorno
		Port:     587,                        // Puerto STARTTLS
		Username: os.Getenv("SMTP_USERNAME"), // IAM SMTP username
		Password: os.Getenv("SMTP_PASSWORD"), // IAM SMTP password
		From:     os.Getenv("SMTP_FROM"),     // Email verificado en SES
	}

	log.Printf("Configurando envío de correo a: %s", email)
	log.Printf("Configuración SMTP - Host: %s, Usuario: %s, From: %s",
		config.Host, config.Username, config.From)

	// Validar configuración pero no fallar en desarrollo
	if config.Host == "" || config.Username == "" || config.Password == "" || config.From == "" {
		errMsg := "Configuración SMTP incompleta. Verifica las variables de entorno SMTP_*"
		magicLink := fmt.Sprintf("http://%s/api/auth/validate?token=%s", host, token)
		log.Printf("%s. Enlace mágico para %s: %s", errMsg, email, magicLink)

		// En desarrollo, intentar continuar con valores por defecto
		if os.Getenv("ENV") == "development" {
			log.Println("Modo desarrollo: Intentando continuar con configuración por defecto")
			// Usar valores por defecto si no están configurados
			if config.Host == "" {
				config.Host = "email-smtp.us-east-2.amazonaws.com"
			}
			if config.Username == "" {
				config.Username = os.Getenv("SMTP_USERNAME")
			}
			if config.Password == "" {
				config.Password = os.Getenv("SMTP_PASSWORD")
			}
			if config.From == "" {
				config.From = os.Getenv("SMTP_FROM")
			}
		} else {
			return fmt.Errorf(errMsg)
		}
	}

	// Crear mensaje
	m := gomail.NewMessage()
	m.SetHeader("From", config.From)
	m.SetHeader("To", email)
	m.SetHeader("Subject", "Tu enlace de inicio de sesión")

	// Cuerpo del correo con estilos
	magicLink := fmt.Sprintf("http://%s/api/auth/validate?token=%s", host, token)
	appName := os.Getenv("APP_NAME")
	if appName == "" {
		appName = "Toolbox API"
	}

	body := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
	    <meta charset="UTF-8">
	    <meta name="viewport" content="width=device-width, initial-scale=1.0">
	    <title>Inicio de sesión - %s</title>
	    <style>
	        body {
	            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, 'Open Sans', 'Helvetica Neue', sans-serif;
	            line-height: 1.6;
	            color: #333;
	            max-width: 600px;
	            margin: 0 auto;
	            padding: 20px;
	        }
	        .header {
	            text-align: center;
	            padding: 20px 0;
	            border-bottom: 2px solid #4F46E5;
	            margin-bottom: 20px;
	        }
	        .logo {
	            color: #4F46E5;
	            font-size: 24px;
	            font-weight: 700;
	            text-decoration: none;
	        }
	        .button {
	            display: inline-block;
	            padding: 12px 24px;
	            background-color: #4F46E5;
	            color: white !important;
	            text-decoration: none;
	            border-radius: 6px;
	            font-weight: 500;
	            margin: 20px 0;
	        }
	        .content {
	            background-color: #f9fafb;
	            padding: 30px;
	            border-radius: 8px;
	            margin: 20px 0;
	        }
	        .footer {
	            text-align: center;
	            margin-top: 30px;
	            color: #6b7280;
	            font-size: 14px;
	        }
	        .code {
	            background-color: #f3f4f6;
	            padding: 10px;
	            border-radius: 4px;
	            font-family: monospace;
	            word-break: break-all;
	        }
	    </style>
	</head>
	<body>
	    <div class="header">
	        <a href="http://%s" class="logo">%s</a>
	    </div>
	    
	    <div class="content">
	        <h2>¡Hola!</h2>
	        <p>Hemos recibido una solicitud para iniciar sesión en tu cuenta de %s.</p>
	        
	        <div style="text-align: center; margin: 30px 0;">
	            <a href="%s" class="button">Iniciar sesión</a>
	        </div>
	        
	        <p>O copia y pega esta URL en tu navegador:</p>
	        <div class="code">%s</div>
	        
	        <p><strong>Importante:</strong> Este enlace es válido por 1 hora. Si no has solicitado este enlace, puedes ignorar este mensaje.</p>
	    </div>
	    
	    <div class="footer">
	        <p>© %d %s. Todos los derechos reservados.</p>
	        <p>Este es un correo automático, por favor no respondas a este mensaje.</p>
	    </div>
	</body>
	</html>
	`, appName, host, appName, appName, magicLink, magicLink, time.Now().Year(), appName)

	m.SetBody("text/html", body)

	// Configurar cliente SMTP para Amazon SES
	log.Printf("Configuración del cliente SMTP con autenticación")
	d := gomail.NewDialer(config.Host, config.Port, config.Username, config.Password)

	// Configuración de depuración
	d.TLSConfig = &tls.Config{
		InsecureSkipVerify: true, // Solo para depuración, NO usar en producción
		ServerName:         config.Host,
	}

	// Log de configuración
	log.Printf("Configurando dialer SMTP - Host: %s, Port: %d, Username: %s",
		config.Host, config.Port, config.Username)

	d.SSL = false               // Usar STARTTLS
	d.LocalName = "toolbox-api" // HELO/EHLO identity

	// Enviar correo
	log.Printf("Iniciando envío de correo a %s a través de %s:%d", email, config.Host, config.Port)

	if err := d.DialAndSend(m); err != nil {
		errMsg := fmt.Sprintf("Error al enviar correo a %s: %v", email, err)
		log.Printf("%s. Detalles: Host=%s, Username=%s, From=%s",
			errMsg, config.Host, config.Username, config.From)
		return fmt.Errorf(errMsg)
	}

	log.Printf("Correo enviado exitosamente a %s", email)
	return nil
}
