const {createMockServer} = require("grpc-mock");
const mockServer = createMockServer({
  protoPath: "/proto/greeter.proto",
  packageName: "greeter",
  serviceName: "Greeter",
  rules: [
    { method: "hello", input: { message: "Hello" }, output: { message: "Hello" } },
    { method: "hello", input: { message: "Hi" }, output: { message: "A little familar, are't you" } },
    { method: "goodbye", input: ".*", output: { message: "Goodbye" } },
    
    {
      method: "howAreYou",
      streamType: "client",
      stream: [
        { input: { message: "Hi" } },
        { input: { message: "How are you?" } },
      ],
      output: { message: "I'm fine, thank you" }
    },

    {
      method: "howAreYou",
      streamType: "client",
      stream: [
        { input: { message: "Hello" } },
        { input: { message: "Do you think it will rain?" } },
      ],
      output: { message: "It does look a bit cloudy" }
    },
    
    {
      method: "niceToMeetYou",
      streamType: "server",
      stream: [
        { output: { message: "Hi, I'm Sana" } },
        { output: { message: "Nice to meet you too" } },
      ],
      input: { message: "Hi. I'm John. Nice to meet you" }
    },

    {
      method: "niceToMeetYou",
      streamType: "server",
      stream: [
        { output: { message: "Hi, I'm Sana" } },
        { output: { message: "Have you met John?" } },
      ],
      input: { message: "Hi. I'm Frank" }
    },
    
    {
      method: "chat",
      streamType: "mutual",
      stream: [
        { input: { message: "Hi" }, output: { message: "Hi there" } },
        { input: { message: "How are you?" }, output: { message: "I'm fine, thank you." } },
      ]
    },

    {
      method: "chat",
      streamType: "mutual",
      stream: [
        { input: { message: "Hello" }, output: { message: "G'day" } },
        { input: { message: "What are you doing today?" }, output: { message: "Chatting with you" } },
      ]
    },
    
    { method: "returnsError", input: { }, error: { code: 3, message: "Message text is required"} },
    
    {
      method: "returnsErrorWithMetadata",
      streamType: "server",
      input: { },
      error: { code: 3, message: "Message text is required", metadata: { key: "value"}}
    }
  ]
});
process.on('SIGINT', function() {
    console.log("Caught interrupt signal");
    process.exit();
});
mockServer.listen("0.0.0.0:50051");