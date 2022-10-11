# Progetto_SDCC
Progetto per il corso di Sistemi Distribuiti e Cloud Computing per la laurea magistrale in Ingegneria Informatica dell'Università di Roma Tor Vergata<br>

## Installazione
Dopo aver scaricato la repository, si può avviare il programma usando lo script [launch.sh](launch.sh)
<br>Questo script può essere configurato con diversi flag:<br>
- flag '-n' per specificare il numero di peer
- flag '-a' per specificare l'algoritmo da usare
- flag '-v' per modalità verbose
- flag '-d' per specificare la congestione di rete
<br><br>L'algoritmo può essere **auth**, **token** o **quorum** e saranno usati rispettivamente l'algorimo di Ricart Agrawala, un algoritmo basato su token centralizzato o l'algoritmo di Maekawa
<br>Il flag verbose creerà dei file di log situati nella cartella */logs*
<br>La velocità della rete può essere specificata come **fast**, **medium** o **slow**<br>

## Spegnimento
Quando si desidera fermare l'esecuzione, avviare lo script [down.sh](down.sh)
<br>Questo script spegnerà tutti i peer assicurandosi di non interromperli prima che la sezione critica venga rilasciata<br>

## Testing
Per testare il funzionamento, si può avviare lo script [testing.sh](test/testing.sh)
<br>Questo script permette di effettuare un test in diverse condizioni specificabili tramite flag:<br>
- flag '-n' per specificare se usare un solo peer o più peer contemporaneamente
- flag '-a' per specificare l'algoritmo da usare
- flag '-d' per specificare la congestione di rete
<br><br>Per il numero di peer si può usare **one** o **many**
<br>L'algoritmo può essere **auth**, **token** o **quorum** e saranno usati rispettivamente l'algorimo di Ricart Agrawala, un algoritmo basato su token centralizzato o l'algoritmo di Maekawa
<br>La velocità della rete può essere specificata come **fast**, **medium** o **slow**

