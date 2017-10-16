# Proxyscript

Programme retournant un fichier particulier en fonction de l'ip appelant.

L'ensemble des fichiers de configuration (.csv, .pac et .toml) sont chargés en mémoire et rechargés en cas de modification.

Ce programme est utilisé pour fournir un fichier proxy.pac en fonction de l'adresse ip du client. Ceci permet par exemple de spécifier un proxy particulier pour un domaine ou un réseau cible particulier et cela que pour un groupe de machine d'un réseau bien identifier.

## Installation

```
go build -o proxyscript

```
Un script d'installation et un fichier systemd sont présents dans le répertoire script.

## Configuration

Modifier le fichier proxyscript.toml pour spécifier l'ip et le port d'écoute du programme ***proxyscript***.

Créer un fichier csv pour associer les fichiers .pac avec un réseau.
Il faut une entrée par ligne, la première ligne qui correspondra au réseau de l'adresse ip du poste client retournera le contenu du fichier .pac correspondant.
