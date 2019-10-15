package helpers

import (
	"database/sql"
	"fmt"
	"errors"

	_ "github.com/denisenkom/go-mssqldb"
)

const connectionString = "your_connection_string_goes_here"

var sqldb *sql.DB = nil

type dataBaseInterface interface {
	Init() error
	Close()
	GetUsers() error
	AddUser(u *UserType) error
	UpdateUser(u *UserType) error
	RemoveNode(nodeID uint32) error
	AddNode(n *NodeType) error
}

type DataBaseType struct {
	_ dataBaseInterface
}

var DB DataBaseType

func (d DataBaseType) UpdateUser(u *UserType) error {
	str := fmt.Sprintf("update tblUsers set TelegramUserName = '%s', TelegramFirstName ='%s', TelegramLastName = '%s' where ID = %v",
		u.UserName, u.FirstName, u.LastName, u.ID)
	_, err := sqldb.Exec(str)
	return err
}

func (d DataBaseType) AddNode(n *NodeType) error {
	str := fmt.Sprintf("insert into tblNodes values (%v, '%s', '%s', 0)", n.UserID, n.BalancesKey, n.NodesKey)
	_, err := sqldb.Exec(str)
	if err != nil {
		fmt.Printf("Error adding user's %v node to DB: %s\n\r", n.UserID, err)
		return err
	}
	rows, err := sqldb.Query("select top 1 ID from tblNodes order by ID desc")
	if err != nil {
		fmt.Println(err)
		return err
	}
	var nodeID uint32
	for rows.Next() {
		err := rows.Scan(&nodeID)
		if err == nil {
			n.ID = nodeID
		}
	}
	rows.Close()
	return nil
}

func (d DataBaseType) AddUser(u *UserType) error {
	str := fmt.Sprintf("insert into tblUsers values (%v, '%s', '%s', '%s', 0)", u.TelegramID, u.UserName, u.FirstName, u.LastName)
	_, err := sqldb.Exec(str)
	if err != nil {
		fmt.Printf("Error adding user to DB: %s\n\r", err)
		return err
	}
	rows, err := sqldb.Query("select top 1 ID from tblUsers order by ID desc")
	if err != nil {
		fmt.Println(err)
		return err
	}
	nodes := make([]*NodeType, 0)
	u.Nodes = &nodes
	var userID uint32
	for rows.Next() {
		err := rows.Scan(&userID)
		if err == nil {
			u.ID = userID
		}
	}
	rows.Close()
	return nil
}

func (d DataBaseType) GetUsers() {
	rows, err := sqldb.Query("select * from tblUsers")
	if err != nil {
		fmt.Printf("Error fetching users from DB: %s\n\r", err)
		return
	}
	var (
		userID		uint32
		tgID		uint32
		tgUserName	string
		tgFirstName string
		tgLastName  string
		isAdmin 	bool
	)
	for rows.Next() {
		err := rows.Scan(&userID, &tgID, &tgUserName, &tgFirstName, &tgLastName, &isAdmin)
		if err == nil {
			user := UserType {
				ID: 		userID,
				TelegramID: tgID,
				UserName:	tgUserName,
				FirstName: 	tgFirstName,
				LastName: 	tgLastName,
				IsAdmin: 	isAdmin,
			}
			nodes := make([]*NodeType, 0)
			user.Nodes = &nodes
			rows2, err2 := sqldb.Query(fmt.Sprintf("select * from tblNodes where (UserID = %v) and (Deleted = 0)", userID))
			if err2 != nil {
				fmt.Printf("Error fetching nodes for user ID %v from DB: %s\n\r", userID, err2)
				continue
			}
			var (
				nodeID	uint32
				bKey	string
				nKey	string
				deleted bool
			)
			for rows2.Next() {
				err2 := rows2.Scan(&nodeID, &userID, &bKey, &nKey, &deleted)
				if err2 == nil {
					node := NodeType {
						ID:	nodeID,
						UserID: userID,
						BalancesKey: bKey,
						NodesKey: nKey,						
					}
					*user.Nodes = append(*user.Nodes, &node)
				}
			}
			rows2.Close()
			Users = append(Users, &user)
		}
	}
	rows.Close()
}

func (d DataBaseType) Init() error {
	var err error = nil
	sqldb, err = sql.Open("mssql", connectionString)
	if err != nil {
		fmt.Println(err)
		return err
	}
	d.GetUsers()
	return nil
}

func (d DataBaseType) Close() {
	sqldb.Close()
}

func (d DataBaseType) RemoveNode(nodeID uint32) error {
	var owner *[]*NodeType = nil
	var idx int
	for _, u := range Users {
		for in, n := range *u.Nodes {
			if n.ID == nodeID {
				owner = u.Nodes
				idx = in
				break
			}
		}
	}
	if owner == nil {
		return errors.New("Node not found")
	}
	str := fmt.Sprintf("update tblNodes set Deleted = 1 where ID = %v", nodeID)
	_, err := sqldb.Exec(str)
	if err != nil {
		fmt.Println("Error removing node from DB")
		return err
	}
	deref := *owner
	*owner = append(deref[:idx], deref[idx+1:]...)
	return nil
}
