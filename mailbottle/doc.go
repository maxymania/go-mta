/*
 * Copyright(C) 2015 Simon Schmidt
 * 
 * This Source Code Form is subject to the terms of the
 * Mozilla Public License, v. 2.0. If a copy of the MPL
 * was not distributed with this file, You can obtain one at
 * http://mozilla.org/MPL/2.0/.
 */

/*
 MailBottle is a both, a Format and a Protocol designed, to store
 mails in Mail-Queue and to transfer it from the SMTP daemon to the
 Queue daemon.

 Other than LMTP, the MailBottle Protocol is designed, to have a Queue.
 Other than LMTP and SMTP, MailBottle is both Protocol and (on disk) Format.
 Other than LMTP and SMTP, MailBottle is designed to use TCP/IP efficiently
 by bundling more information into a single Request and by enabling Streaming
 transfers.
*/
package mailbottle

