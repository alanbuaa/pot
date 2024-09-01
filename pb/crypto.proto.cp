syntax = "proto3";
package pb;

option go_package="./pb";

message SRS{

}

message PointG1{

}

message EncShare{

}

message SchnorrProof{

}

message DrawProof{

}

message PointG1List{
  repeated PointG1 PointG1s = 1;
}

message EncSharelist{
  repeated EncShare EncShares = 1;
}

message CryptoElement{
  SRS SRS = 1;
  repeated PointG1List PvssPkLists = 2;
  repeated PointG1List ShareCommitLists = 3;
  repeated PointG1List CoeffCommitLists = 4;
  repeated EncSharelist EncSharesLists = 5;
  repeated PointG1List CommitteePKLists = 6
  SchnorrProof SrsUpdateProof = 7;
  DrawProof ShuffleProof = 8;
  DrawProof DrawProof = 9;
  uint64 CommitteeWorkHeightList =10;
}
