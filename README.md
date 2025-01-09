# sensonet

sensonet is a library that provides functions to read data from Vaillant heating systems, especially heat pumps, and to initiate certain routines on these systems.
The communication works via the myVaillant portal and the sensonet module (VR921). So you need a Vaillant heating system with a VR921 module and a myVaillant user account. 
(Presumably the library also works with a VR940f instead of a VR921.)

## Features
- Initalisation of communication to the user account provided 
- Reading which "homes" are available under the user account
- Reading the system information for a selected systemId consisting of configuration data, property data and state data 
- Reading the device information for a selected systemId
- Reading the historical energy data for selected devices 
- Reading the current power consumption for selected systemId and underlying devices (this is unfortunately not supported by all heating systems) 
- Starting and stopping of hotwater boosts and of zone quick veto sessions
- Starting and stopping of strategy based quick mode sessions
- Data read from the myVaillant portal are cached by in a controller object to limit the number of http requests to the portal. The usage of this controller and
  its methods is recommended, but you can also relinquish to use the controller and use the functions (methods of the connection object) that directly do http requests.

## Acknowledgements

Which Vaillant urls to use and which and how data have to be sent to the Vaillant API are partly inspired by the myPyllant project of signalkraft. Many thanks for that! 
 
## Getting Started

This project is still in a preliminary state.

