const onlineUsersDiv = document.getElementById('onlineUsers');
const chatMessages = document.getElementById('messages');
const messageInput = document.getElementById('messageInput');
const sendButton = document.getElementById('sendButton');

let userName = 'Anonymous';

// Prompt the user for a name
const enteredName = window.prompt("Please enter your name:");

if (enteredName!==null&&enteredName.trim() !== '') {
    userName = enteredName;
}

const ws = new WebSocket('ws://localhost:8080/ws');

// Function to update the list of online users
function updateOnlineUsers(users) {
    const onlineUsersLabel = document.createTextNode(`Online Users: ${users}`);
    onlineUsersDiv.innerHTML = '';
    onlineUsersDiv.appendChild(onlineUsersLabel);
}

// Function to display a message in the chat area
function displayMessage(message) {
    const messageDiv = document.createElement('div');
    messageDiv.innerText = message;
    chatMessages.appendChild(messageDiv);
    chatMessages.scrollTop = chatMessages.scrollHeight;
}

// Handle WebSocket open event
ws.onopen = () => {
    // Send the user's name to the server
    ws.send(userName);
};

// Handle WebSocket error event
ws.onerror = (error) => {
    console.error('WebSocket Error:', error);
};

// Handle WebSocket close event
ws.onclose = (event) => {
    if (event.code === 1000) {
        console.log('WebSocket closed normally');
    } else {
        console.error('WebSocket closed with code:', event.code);
    }
};

// Modify the Send button click event
sendButton.addEventListener('click', () => {
    const message = messageInput.value;
    if (message !== '') {
        ws.send(message);
        messageInput.value = '';
        chatMessages.scrollTop = chatMessages.scrollHeight;
    }
});

// Listen for WebSocket messages and display them
ws.addEventListener('message', (event) => {
    const message = event.data;

    if (message.startsWith('Online Users: ')) {
        const users = message.replace('Online Users: ', '');
        updateOnlineUsers(users);
    } else {
        displayMessage(message);
    }
});