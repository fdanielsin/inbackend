package shared

// APIName est le nom public de l'API, celui de info.title du contrat.
const APIName = "InDrive Backend API"

// APIVersion est la version de l'API : la seule valeur du code, lue par /health et
// par le sommaire de la racine. Une release la change avec info.version du contrat
// (un test compare les deux).
const APIVersion = "1.3.0"
