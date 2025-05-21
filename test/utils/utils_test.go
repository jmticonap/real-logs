package utils_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	// Asegúrate de que esta ruta de importación sea correcta para tu proyecto
	"github.com/jmticonap/real-logs/domain"
	"github.com/jmticonap/real-logs/utils"
)

// --- Tests para EnsureDir (sin cambios respecto a la respuesta anterior) ---
func TestEnsureDir(t *testing.T) {
	// Helper para crear un directorio temporal para los tests
	createTempDir := func(t *testing.T) string {
		t.Helper()
		dir, err := os.MkdirTemp("", "testdir_")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		return dir
	}

	t.Run("DirectorioNoExiste", func(t *testing.T) {
		baseDir := createTempDir(t)
		defer os.RemoveAll(baseDir)

		testPath := filepath.Join(baseDir, "newdir")

		err := utils.EnsureDir(testPath)
		if err != nil {
			t.Errorf("EnsureDir() error = %v, wantErr nil", err)
		}

		info, statErr := os.Stat(testPath)
		if os.IsNotExist(statErr) {
			t.Errorf("Directorio no fue creado en %s", testPath)
		} else if !info.IsDir() {
			t.Errorf("Ruta %s fue creada pero no es un directorio", testPath)
		}
	})

	t.Run("DirectorioYaExiste", func(t *testing.T) {
		baseDir := createTempDir(t)
		defer os.RemoveAll(baseDir)

		existingPath := filepath.Join(baseDir, "existingdir")
		if err := os.Mkdir(existingPath, 0755); err != nil {
			t.Fatalf("Fallo al crear directorio preexistente: %v", err)
		}

		err := utils.EnsureDir(existingPath)
		if err != nil {
			t.Errorf("EnsureDir() error = %v, wantErr nil para directorio existente", err)
		}
	})

	t.Run("RutaExistePeroEsUnArchivo", func(t *testing.T) {
		baseDir := createTempDir(t)
		defer os.RemoveAll(baseDir)

		filePath := filepath.Join(baseDir, "file.txt")
		file, err := os.Create(filePath)
		if err != nil {
			t.Fatalf("Fallo al crear archivo de prueba: %v", err)
		}
		file.Close()

		err = utils.EnsureDir(filePath)
		expectedErrorMsg := fmt.Sprintf("%s ya existe pero no es un directorio", filePath)
		if err == nil {
			t.Errorf("EnsureDir() error = nil, wantErr '%s'", expectedErrorMsg)
		} else if err.Error() != expectedErrorMsg {
			t.Errorf("EnsureDir() error = '%v', wantErr '%s'", err, expectedErrorMsg)
		}
	})

	t.Run("ErrorAlCrearDirectorio_RutaInvalida", func(t *testing.T) {
		baseDir := createTempDir(t)
		defer os.RemoveAll(baseDir)

		fileAsPathComponent := filepath.Join(baseDir, "filecomponent")
		f, ferr := os.Create(fileAsPathComponent)
		if ferr != nil {
			t.Fatalf("No se pudo crear el archivo para la prueba: %v", ferr)
		}
		f.Close()

		invalidPath := filepath.Join(fileAsPathComponent, "newdir")
		err := utils.EnsureDir(invalidPath)

		if err == nil {
			t.Errorf("EnsureDir() con ruta inválida %s; error = nil, se esperaba un error", invalidPath)
		}
	})
}

// --- Tests para ParseHour ---
func TestParseHour(t *testing.T) {
	now := time.Now() // Para comparar la fecha en el primer caso

	tests := []struct {
		name      string
		hourStr   string
		wantTime  time.Time
		wantErr   bool
		checkFunc func(t *testing.T, gotTime, wantTime time.Time, gotErr error)
	}{
		{
			name:     "FormatoHHMMValido",
			hourStr:  "15:04",
			wantTime: time.Date(now.Year(), now.Month(), now.Day(), 15, 4, 0, 0, now.Location()),
			wantErr:  false,
		},
		{
			name:    "FormatoYYYYMMDDTHHMMValido",
			hourStr: "2023-10-26T08:30",
			// CORRECCIÓN AQUÍ: time.Parse sin offset devuelve UTC.
			wantTime: time.Date(2023, 10, 26, 8, 30, 0, 0, time.UTC), // Cambiado de time.Local a time.UTC
			wantErr:  false,
		},
		{
			name:    "FormatoInvalido_Letras",
			hourStr: "ab:cd",
			wantErr: true,
		},
		{
			name:    "FormatoInvalido_HoraFueraDeRango",
			hourStr: "25:00",
			wantErr: true,
		},
		{
			name:    "FormatoInvalido_MinutoFueraDeRango",
			hourStr: "10:70",
			wantErr: true,
		},
		{
			name:    "FormatoInvalido_FechaParcialInvalida",
			hourStr: "2023-13-01T10:00",
			wantErr: true,
		},
		{
			name:    "StringVacio",
			hourStr: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTime, gotErr := utils.ParseHour(tt.hourStr) // Asumiendo que ParseHour está en el paquete 'utils' que importas

			if (gotErr != nil) != tt.wantErr {
				t.Errorf("ParseHour() error = %v, wantErr %v", gotErr, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Comparamos usando time.Equal() que es más robusto para zonas horarias
				if !gotTime.Equal(tt.wantTime) {
					t.Errorf("ParseHour() gotTime = %v (%s), want %v (%s)", gotTime, gotTime.Location(), tt.wantTime, tt.wantTime.Location())
				}
			} else {
				if !gotTime.IsZero() {
					t.Errorf("ParseHour() en caso de error, se esperaba tiempo cero, pero se obtuvo %v", gotTime)
				}
			}
		})
	}
}

// --- Tests para GetLogItem (ACTUALIZADO con la estructura real de domain.LogType) ---

func TestGetLogItem(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		wantLog domain.LogType
		wantErr bool
	}{
		{
			name: "JSONValidoCompleto",
			line: `{"level":"INFO","timestamp":"2023-10-26T10:00:00Z","pid":12345,"hostname":"server01","traceId":"trace-abc","spanId":"span-xyz","parentId":"parent-123","msg":"Log message successful"}`,
			wantLog: domain.LogType{
				Level:     "INFO",
				Timestamp: "2023-10-26T10:00:00Z",
				Pid:       12345,
				Hostname:  "server01",
				TraceId:   "trace-abc",
				SpanId:    "span-xyz",
				ParentId:  "parent-123",
				Msg:       "Log message successful",
			},
			wantErr: false,
		},
		{
			name: "JSONValidoConAlgunosCamposFaltantes",
			line: `{"level":"DEBUG","timestamp":"2023-10-27T11:00:00Z","pid":54321,"msg":"Partial log"}`,
			// Los campos string faltantes serán "", int será 0
			wantLog: domain.LogType{
				Level:     "DEBUG",
				Timestamp: "2023-10-27T11:00:00Z",
				Pid:       54321,
				Hostname:  "", // Valor cero para string
				TraceId:   "", // Valor cero para string
				SpanId:    "", // Valor cero para string
				ParentId:  "", // Valor cero para string
				Msg:       "Partial log",
			},
			wantErr: false, // json.Unmarshal no da error si faltan campos, los deja en su valor cero.
		},
		{
			name:    "JSONInvalido_Malformed",
			line:    `{"level":"ERROR","msg":"Malformed JSON`, // Falta la llave de cierre y comillas
			wantLog: domain.LogType{},                         // Se espera el valor cero de LogType
			wantErr: true,
		},
		{
			name: "JSONConTipoIncorrectoParaCampoNumerico",
			line: `{"level":"WARN","timestamp":"2023-10-28T12:00:00Z","pid":"not-a-number","msg":"PID type error"}`,
			// json.Unmarshal devolverá un error porque "not-a-number" no puede ser parseado a int para Pid.
			wantLog: domain.LogType{}, // Se espera el valor cero de LogType
			wantErr: true,
		},
		{
			name: "JSONConTipoIncorrectoParaCampoString",
			line: `{"level":123,"timestamp":"2023-10-28T13:00:00Z","pid":6789,"msg":"Level type error"}`,
			// json.Unmarshal devolverá un error porque 123 no puede ser parseado a string para Level.
			wantLog: domain.LogType{}, // Se espera el valor cero de LogType
			wantErr: true,
		},
		{
			name:    "StringVacio",
			line:    "",
			wantLog: domain.LogType{},
			wantErr: true, // json.Unmarshal con string vacío da error (normalmente EOF)
		},
		{
			name: "JSONConSoloCamposDesconocidos",
			line: `{"campo_desconocido":"valor","otro_campo":true}`,
			// json.Unmarshal ignorará los campos desconocidos y no llenará ningún campo de LogType.
			// No se producirá error.
			wantLog: domain.LogType{}, // Todos los campos de LogType quedarán con su valor cero.
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLog, err := utils.GetLogItem(tt.line)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetLogItem() error = %v, wantErr %v", err, tt.wantErr)
				// Si el estado del error es inesperado, imprimir los valores para depuración
				if err != nil {
					t.Logf("Error recibido: %s", err.Error())
				}
				t.Logf("Línea de entrada: %s", tt.line)
				return
			}

			if !tt.wantErr && !reflect.DeepEqual(gotLog, tt.wantLog) {
				t.Errorf("GetLogItem() \ngotLog = %+v, \nwantLog = %+v", gotLog, tt.wantLog)
			}

			if tt.wantErr && !reflect.DeepEqual(gotLog, domain.LogType{}) {
				t.Errorf("GetLogItem() en caso de error, se esperaba LogType cero, pero se obtuvo %+v", gotLog)
			}
		})
	}
}
