package utils_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jmticonap/real-logs/domain"
	"github.com/jmticonap/real-logs/utils"
)

// --- Tests para EnsureDir ---
func TestEnsureDir(t *testing.T) {
	// Helper para crear un directorio temporal para los tests
	createTempDir := func(t *testing.T) string {
		t.Helper()
		dir, err := os.MkdirTemp("", "testdir_")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err) // Fatalf está bien aquí, es setup
		}
		return dir
	}

	t.Run("DirectorioNoExiste", func(t *testing.T) {
		baseDir := createTempDir(t)
		defer os.RemoveAll(baseDir)

		testPath := filepath.Join(baseDir, "newdir")

		err := utils.EnsureDir(testPath)
		assert.NoError(t, err, "EnsureDir() no debería retornar error")
		assert.DirExists(t, testPath, "El directorio debería existir")
	})

	t.Run("DirectorioYaExiste", func(t *testing.T) {
		baseDir := createTempDir(t)
		defer os.RemoveAll(baseDir)

		existingPath := filepath.Join(baseDir, "existingdir")
		err := os.Mkdir(existingPath, 0755)
		require.NoError(t, err, "Fallo al crear directorio preexistente") // require para fallar rápido si el setup falla

		err = utils.EnsureDir(existingPath)
		assert.NoError(t, err, "EnsureDir() no debería retornar error para un directorio existente")
		assert.DirExists(t, existingPath, "El directorio preexistente aún debería existir")
	})

	t.Run("RutaExistePeroEsUnArchivo", func(t *testing.T) {
		baseDir := createTempDir(t)
		defer os.RemoveAll(baseDir)

		filePath := filepath.Join(baseDir, "file.txt")
		file, err := os.Create(filePath)
		require.NoError(t, err, "Fallo al crear archivo de prueba")
		file.Close()

		err = utils.EnsureDir(filePath)
		expectedErrorMsg := fmt.Sprintf("%s ya existe pero no es un directorio", filePath)
		assert.EqualError(t, err, expectedErrorMsg, "EnsureDir() debería retornar un error específico")
	})

	t.Run("ErrorAlCrearDirectorio_RutaInvalida", func(t *testing.T) {
		baseDir := createTempDir(t)
		defer os.RemoveAll(baseDir)

		fileAsPathComponent := filepath.Join(baseDir, "filecomponent")
		f, ferr := os.Create(fileAsPathComponent)
		require.NoError(t, ferr, "No se pudo crear el archivo para la prueba")
		f.Close()

		invalidPath := filepath.Join(fileAsPathComponent, "newdir")
		err := utils.EnsureDir(invalidPath)
		assert.Error(t, err, "EnsureDir() con ruta inválida debería retornar un error")
		// Opcionalmente, puedes ser más específico sobre el tipo de error si es posible y relevante
		// assert.Contains(t, err.Error(), "not a directory") // O un mensaje similar del sistema operativo
	})
}

// --- Tests para ParseHour ---
func TestParseHour(t *testing.T) {
	now := time.Now() // Para comparar la fecha en el primer caso

	tests := []struct {
		name     string
		hourStr  string
		wantTime time.Time
		wantErr  bool
	}{
		{
			name:     "FormatoHHMMValido",
			hourStr:  "15:04",
			wantTime: time.Date(now.Year(), now.Month(), now.Day(), 15, 4, 0, 0, now.Location()),
			wantErr:  false,
		},
		{
			name:     "FormatoYYYYMMDDTHHMMValido",
			hourStr:  "2023-10-26T08:30",
			wantTime: time.Date(2023, 10, 26, 8, 30, 0, 0, time.UTC),
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
			gotTime, gotErr := utils.ParseHour(tt.hourStr)

			if tt.wantErr {
				assert.Error(t, gotErr, "ParseHour() debería retornar un error")
				assert.True(t, gotTime.IsZero(), "ParseHour() en caso de error, se esperaba tiempo cero")
			} else {
				assert.NoError(t, gotErr, "ParseHour() no debería retornar error")
				// Usamos time.Equal() para comparar tiempos correctamente, especialmente con zonas horarias.
				// assert.Equal no siempre funciona bien con time.Time si las ubicaciones son diferentes instancias
				// aunque representen la misma zona horaria.
				assert.True(t, tt.wantTime.Equal(gotTime), "ParseHour() gotTime = %v (%s), want %v (%s)", gotTime, gotTime.Location(), tt.wantTime, tt.wantTime.Location())
			}
		})
	}
}

// --- Tests para GetLogItem ---
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
			wantLog: domain.LogType{
				Level:     "DEBUG",
				Timestamp: "2023-10-27T11:00:00Z",
				Hostname:  "",
				TraceId:   "",
				SpanId:    "",
				ParentId:  "",
				Msg:       "Partial log",
			},
			wantErr: false,
		},
		{
			name:    "JSONInvalido_Malformed",
			line:    `{"level":"ERROR","msg":"Malformed JSON`,
			wantLog: domain.LogType{},
			wantErr: true,
		},
		{
			name:    "JSONConTipoIncorrectoParaCampoNumerico",
			line:    `{"level":"WARN","timestamp":"2023-10-28T12:00:00Z","pid":"not-a-number","msg":"PID type error"}`,
			wantLog: domain.LogType{},
			wantErr: true,
		},
		{
			name:    "JSONConTipoIncorrectoParaCampoString",
			line:    `{"level":123,"timestamp":"2023-10-28T13:00:00Z","pid":6789,"msg":"Level type error"}`,
			wantLog: domain.LogType{},
			wantErr: true,
		},
		{
			name:    "StringVacio",
			line:    "",
			wantLog: domain.LogType{},
			wantErr: true,
		},
		{
			name:    "JSONConSoloCamposDesconocidos",
			line:    `{"campo_desconocido":"valor","otro_campo":true}`,
			wantLog: domain.LogType{}, // Se espera que los campos de LogType queden con su valor cero.
			wantErr: false,            // json.Unmarshal ignora campos desconocidos por defecto.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLog, err := utils.GetLogItem(tt.line)

			if !tt.wantErr {
				assert.NoError(t, err, "GetLogItem() no debería retornar error")
				assert.Equal(t, tt.wantLog, gotLog, "GetLogItem() el log parseado no es el esperado")
			}
		})
	}
}

func TestGetAllFilesRecursive(t *testing.T) {
	t.Run(
		"Should make a list with all files of the dir.",
		func(t *testing.T) {
			files, err := utils.GetAllFilesRecursive("./dir-test")
			expected := []string{
				"dir-test/dir-inner/txt2.txt",
				"dir-test/txt1.txt",
			}
			assert.NoError(t, err, "It shoult be no error")
			assert.Equal(t, files, expected)
		},
	)

	t.Run(
		"Should return an error if the dir does not exist",
		func(t *testing.T) {
			_, err := utils.GetAllFilesRecursive("./worng-dir")

			assert.Error(t, err)
		},
	)
}

func TestGetPerformanceLogInfo(t *testing.T) {
	t.Run(
		"Should get the array of performance information",
		func(t *testing.T) {
			log := domain.LogType{
				Level:     "INFO",
				Timestamp: "2025-05-19T12:23:57.262-05:00",
				Hostname:  "se-core-charge-565b6d5fd4-xvqg8",
				TraceId:   "2fa1c5be-146d-46ae-a028-95bc160fe373",
				SpanId:    "70b525fa-e495-4cac-8a4c-ac0bbfd6556f",
				ParentId:  "2fa1c5be-146d-46ae-a028-95bc160fe373",
				Msg:       "{\n  title: 'Performance Log',\n  performanceInfo: [\n    {\n      exectime: 3.511593,\n      origin: 'RedisEcommerceRepository',\n      method: 'getMerchant',\n      percentage: '0.08 %'\n    },\n    {\n      exectime: 2.260096,\n      origin: 'RedisEcommerceRepository',\n      method: 'getProducts',\n      percentage: '0.05 %'\n    },\n    {\n      exectime: 403.695438,\n      origin: 'LambdaCorePciAdapter',\n      method: 'verifyToken',\n      percentage: '9.3 %'\n    },\n    {\n      exectime: 71.877858,\n      origin: 'HttpBusinessMerchantTrxAdapter',\n      method: 'getBinById',\n      percentage: '1.66 %'\n    },\n    {\n      exectime: 2.60972,\n      origin: 'RedisEcommerceRepository',\n      method: 'getProducts',\n      percentage: '0.06 %'\n    },\n    {\n      exectime: 2.431582,\n      origin: 'RedisEcommerceRepository',\n      method: 'getCurrencies',\n      percentage: '0.06 %'\n    },\n    {\n      exectime: 2.011132,\n      origin: 'RedisEcommerceRepository',\n      method: 'getAntifraud',\n      percentage: '0.05 %'\n    },\n    {\n      exectime: 171.497673,\n      origin: 'LambdaCorePciAdapter',\n      method: 'updateChargeRefId',\n      percentage: '3.95 %'\n    },\n    {\n      exectime: 3222.982197,\n      origin: 'LambdaFisAdapter',\n      method: 'authorizeCharge',\n      percentage: '74.22 %'\n    },\n    {\n      exectime: 35.949566,\n      origin: 'HttpCoreCommissionsAdapter',\n      method: 'getCommission',\n      percentage: '0.83 %'\n    },\n    {\n      exectime: 142.813182,\n      origin: 'PgChargeRepository',\n      method: 'add',\n      percentage: '3.29 %'\n    },\n    {\n      exectime: 4342.263772,\n      origin: 'ChargeController',\n      method: 'createCharge',\n      percentage: '100 %'\n    }\n  ]\n}",
			}
			expected := []domain.PerformanceType{
				{
					Exectime:   3.511593,
					Origin:     "RedisEcommerceRepository",
					Method:     "getMerchant",
					Percentage: "0.08 %",
				},
				{
					Exectime:   2.260096,
					Origin:     "RedisEcommerceRepository",
					Method:     "getProducts",
					Percentage: "0.05 %",
				},
				{
					Exectime:   403.695438,
					Origin:     "LambdaCorePciAdapter",
					Method:     "verifyToken",
					Percentage: "9.3 %",
				},
				{
					Exectime:   71.877858,
					Origin:     "HttpBusinessMerchantTrxAdapter",
					Method:     "getBinById",
					Percentage: "1.66 %",
				},
				{
					Exectime:   2.60972,
					Origin:     "RedisEcommerceRepository",
					Method:     "getProducts",
					Percentage: "0.06 %",
				},
				{
					Exectime:   2.431582,
					Origin:     "RedisEcommerceRepository",
					Method:     "getCurrencies",
					Percentage: "0.06 %",
				},
				{
					Exectime:   2.011132,
					Origin:     "RedisEcommerceRepository",
					Method:     "getAntifraud",
					Percentage: "0.05 %",
				},
				{
					Exectime:   171.497673,
					Origin:     "LambdaCorePciAdapter",
					Method:     "updateChargeRefId",
					Percentage: "3.95 %",
				},
				{
					Exectime:   3222.982197,
					Origin:     "LambdaFisAdapter",
					Method:     "authorizeCharge",
					Percentage: "74.22 %",
				},
				{
					Exectime:   35.949566,
					Origin:     "HttpCoreCommissionsAdapter",
					Method:     "getCommission",
					Percentage: "0.83 %",
				},
				{
					Exectime:   142.813182,
					Origin:     "PgChargeRepository",
					Method:     "add",
					Percentage: "3.29 %",
				},
				{
					Exectime:   4342.263772,
					Origin:     "ChargeController",
					Method:     "createCharge",
					Percentage: "100 %",
				},
			}
			perform, err := utils.GetPerformanceLogInfo(log)

			assert.NoError(t, err, "Shout no return an error")
			assert.Equal(t, perform, expected)
		},
	)
}
