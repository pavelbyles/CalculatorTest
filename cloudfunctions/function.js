/**
 * Triggered from a message on a Cloud Pub/Sub topic.
 *
 * @param {!Object} event The Cloud Functions event.
 * @param {!Function} The callback function.
 */
exports.writeCalcResult = function (event, callback) {
  const pubsubMessage = event.data;
  const content = pubsubMessage.data ? Buffer.from(pubsubMessage.data, 'base64').toString() : '{"name":"World"}';
  console.log('Input is, %s', content);
  const parsedContent = JSON.parse(content)
  console.log('Hello, %s', parsedContent.name);

  callback();
};
