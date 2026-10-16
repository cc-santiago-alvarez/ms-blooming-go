package shared_domain

// Claves del catálogo de textos. Son las que se guardan en bondHistory.text y activity.description.
const (
	TextBondAdded                        = "BOND_ADDED"
	TextBondUpdated                      = "BOND_UPDATED"
	TextBondUpdatedKey                   = "BOND_UPDATED_KEY"
	TextBondRemove                       = "BOND_REMOVE"
	TextBondBoundSold                    = "BOND_BOUND_SOLD"
	TextBondBoundPurchased               = "BOND_BOUND_PURCHASED"
	TextBondCreated                      = "BOND_CREATED"
	TextBondBoundSubjectPurchased        = "BOND_BOUND_SUBJECT_PURCHASED"
	TextBondBoundMessagePurchased        = "BOND_BOUND_MESSAGE_PURCHASED"
	TextBondBoundSubjectSold             = "BOND_BOUND_SUBJECT_SOLD"
	TextBondBoundMessageSold             = "BOND_BOUND_MESSAGE_SOLD"
	TextBondBoundSubjectTook             = "BOND_BOUND_SUBJECT_TOOK"
	TextBondBoundMessageTook             = "BOND_BOUND_MESSAGE_TOOK"
	TextBondBoundMessageMaturity         = "BOND_BOUND_MESSAGE_MATURITY"
	TextBondBoundMessageMaturitySucess   = "BOND_BOUND_MESSAGE_MATURITY_SUCESS"
	TextBondBoundSubjectMassivePurchased = "BOND_BOUND_SUBJECT_MASSIVE_PURCHASED"
	TextBondBoundMessageMassivePurchased = "BOND_BOUND_MESSAGE_MASSIVE_PURCHASED"
	TextBalanceAccountWithdrawForBonus   = "BALANCE_ACCOUNT_WITHDRAW_FOR_BONUS"
	TextBalanceAccountWithdrawContent    = "BALANCE_ACCOUNT_WITHDRAW_CONTENT"
	TextNewWithdrawal                    = "NEW_WITHDRAWAL"
	TextNotExist                         = "NOT_EXIST"
	TextFilenameReport                   = "FILENAME_REPORT"
)

// texts son las plantillas del legado, con placeholders {0}, {1}, … para Format.
var texts = map[string]string{
	TextBondAdded:                 "Bono Añadido",
	TextBondUpdated:               "Bono Actualizado",
	TextBondUpdatedKey:            "Key Actualizado",
	TextBondRemove:                "Bono Removido",
	TextBondBoundSold:             "Bono Vendido",
	TextBondBoundPurchased:        "Bono Comprado",
	TextBondCreated:               "Se ha creado el bono <b>{0}</b> con un valor de <b>{1}</b>, no te pierdas de esta oportunidad.",
	TextBondBoundSubjectPurchased: "Compra de Bono ({0}), por $ {1}",
	TextBondBoundMessagePurchased: "Gracias, <b>{0} {1}</b>, has adquirido el bono <b>{2}</b> por un monto total de $ {3}<br/></br>El sistema automaticamente le descontara el dinero de la cuenta <b>{4}</b>. Dicho dinero será descontado de su nómina.",
	TextBondBoundSubjectSold:      "Bono {0} vendido",
	TextBondBoundMessageSold: `
  <h3>Detalles del Bono Vendido</h3>
  <strong>Bono:</strong> {0} <br>
  <strong>Key:</strong> {1} <br>
  <strong>Valor:</strong> {2} <br>
  <strong>Crecimiento:</strong> {3} % <br>
  <strong>Maduración:</strong> {4} días <br>
  <strong>Fecha de compra:</strong> {5} <br>
  <strong>Fecha de maduración:</strong> {6} <br>
  <hr>
  <h4>Se ha generado la siguiente consignación</h4>
  <strong>Cuenta:</strong> {7} <br>
  <strong>Ganancia:</strong> {8} <br>
  <strong>Monto:</strong> {9} <br>
  <strong>Disponible Actual:</strong> {10} <br>
`,
	TextBondBoundSubjectTook: "Bono {0} canjeado",
	TextBondBoundMessageTook: `
  <h3>Detalles del Bono Canjeado</h3>
  <strong>Bono:</strong> {0} <br>
  <strong>Key:</strong> {1} <br>
  <strong>Valor:</strong> {2} <br>
  <strong>Crecimiento:</strong> {3} % <br>
  <strong>Maduración:</strong> {4} días <br>
  <strong>Fecha de compra:</strong> {5} <br>
  <strong>Fecha de maduración:</strong> {6} <br>
  <hr>
  <h4>Se ha generado la siguiente consignación</h4>
  <strong>Cuenta:</strong> {7} <br>
  <strong>Ganancia:</strong> {8} <br>
  <strong>Monto:</strong> {9} <br>
  <strong>Disponible Actual:</strong> {10} <br>
`,
	TextBondBoundMessageMaturity: "Bono Madurado",
	TextBondBoundMessageMaturitySucess: `
  Estimado <strong>{0} {1}</strong>, un cordial saludo, el siguiente mensaje es para notificarle que los siguientes bonos han madurado:
  <br>{2}<br>
  Entra y canjea o vende tus bonos nuevamente, gracias por su compra.
`,
	TextBondBoundSubjectMassivePurchased: "Compra de {0} Bonos, por $ {1}",
	TextBondBoundMessageMassivePurchased: `
Felicidades <strong>{0}</strong>, has adquirido <strong>{1}</strong> bonos por un monto total de $ <strong>{2}</strong> con un valor de $ <strong>{3}</strong> c/u. 
El sistema automaticamente le descontara el dinero de la cuenta <strong>{4}</strong>. Dicho dinero será descontado de su nómina.<br/><br/>
<table cellpadding="9px" cellspacing="0" border="0">
<thead>
<tr>
<th style="border: solid 1px #d8d8d8;">#</th>
<th style="border: solid 1px #d8d8d8;">Bono</th>
<th style="border: solid 1px #d8d8d8;"><small>BALANCE</small><br>Código Ref</th>
<th style="border: solid 1px #d8d8d8;">Valor</th>
</tr>
</thead>
<tbody>{5}</tbody>
</table>
`,
	TextBalanceAccountWithdrawForBonus: "Señor(a) <strong>{0}</strong>, por medio de la presente le informamos que se ha realizado el descuento automatico de su cuenta número <strong>{1}</strong> por un monto de <strong>$ {2}</strong> por la compra de los siguentes bonos: <strong>{3}</strong>; con un valor de <strong>$ {4}</strong> c/u.",
	TextBalanceAccountWithdrawContent:  "Señor(a) <strong>{0}</strong>, por medio de la presente le informamos que se ha realizado un retiro de su cuenta número <strong>{1}</strong> por un monto de <strong>$ {2}</strong> en nuestro sistema de BALANCE.",
	TextNewWithdrawal:                  "Nuevo Retiro",
	TextNotExist:                       "Disculpe, {0} no existe.",
	TextFilenameReport:                 "reporte-{0}_{1}",
}

// Text devuelve la plantilla de key, o la clave misma si no está en el catálogo.
func Text(key string) string {
	if tpl, ok := texts[key]; ok {
		return tpl
	}
	return key
}
