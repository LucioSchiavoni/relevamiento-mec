package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type EquipoInfo struct {
	FechaRelevamiento string
	ComputerName      string
	NombreAnterior    string
	MacAddress        string
	IPAddress         string
	Piso              string
	Oficina           string
}

type EquipoResult struct {
	Success      bool
	InsertedID   int64
	RowsAffected int64
	VerifiedData *EquipoVerificado
	ErrorMessage string
}

type EquipoVerificado struct {
	ID           int64
	ComputerName string
	IPAddress    string
	MacAddress   string
	Oficina      string
	Piso         string
}

func CreateEquiposRepository(db *sql.DB, equipo EquipoInfo) (*EquipoResult, error) {
	result := &EquipoResult{
		Success: false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Error iniciando transaccion: %v", err)
		return result, err
	}

	defer func() {
		if !result.Success {
			tx.Rollback()
		}
	}()

	query := `INSERT INTO equipo_info 
		(fecha_relevamiento, computer_name, nombre_anterior, mac_address, ip_address, piso, oficina)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	execResult, err := tx.ExecContext(ctx, query,
		equipo.FechaRelevamiento,
		equipo.ComputerName,
		equipo.NombreAnterior,
		equipo.MacAddress,
		equipo.IPAddress,
		equipo.Piso,
		equipo.Oficina,
	)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Error ejecutando INSERT: %v", err)
		return result, err
	}

	rowsAffected, err := execResult.RowsAffected()
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Error obteniendo rows affected: %v", err)
		return result, err
	}
	result.RowsAffected = rowsAffected

	if rowsAffected == 0 {
		result.ErrorMessage = "No se inserto ningun registro"
		return result, fmt.Errorf("no se inserto ningun registro")
	}

	lastID, err := execResult.LastInsertId()
	if err == nil {
		result.InsertedID = lastID
	}

	verificado, err := verificarInsercion(ctx, tx, equipo)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Error verificando insercion: %v", err)
		return result, err
	}
	result.VerifiedData = verificado

	err = tx.Commit()
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Error en commit: %v", err)
		return result, err
	}

	result.Success = true
	return result, nil
}

func verificarInsercion(ctx context.Context, tx *sql.Tx, equipo EquipoInfo) (*EquipoVerificado, error) {
	verificado := &EquipoVerificado{}

	verifyQuery := `SELECT id, computer_name, ip_address, mac_address, oficina, piso
					FROM equipo_info 
					WHERE computer_name = ? 
					  AND mac_address = ? 
					  AND fecha_relevamiento = ?
					ORDER BY id DESC 
					LIMIT 1`

	err := tx.QueryRowContext(ctx, verifyQuery,
		equipo.ComputerName,
		equipo.MacAddress,
		equipo.FechaRelevamiento,
	).Scan(
		&verificado.ID,
		&verificado.ComputerName,
		&verificado.IPAddress,
		&verificado.MacAddress,
		&verificado.Oficina,
		&verificado.Piso,
	)

	if err != nil {
		return nil, fmt.Errorf("no se pudo verificar el registro insertado: %v", err)
	}

	return verificado, nil
}
